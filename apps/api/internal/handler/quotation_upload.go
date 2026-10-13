package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
	"github.com/madhavr16/QuoteWise/apps/api/internal/service"
)

// QuotationUploadHandler serves the quotation-upload endpoint
// (Milestone 2 part 2). The handler stays thin: parse the multipart
// form, resolve the auth context, stream the file to the service,
// answer with a DTO (ADR-015). Validation, checksums, storage keys,
// and job creation live in the service and store layers.
type QuotationUploadHandler struct {
	quotations *service.QuotationService
	logger     *slog.Logger
}

// NewQuotationUploadHandler wires the quotation-upload endpoint.
func NewQuotationUploadHandler(quotations *service.QuotationService, logger *slog.Logger) *QuotationUploadHandler {
	return &QuotationUploadHandler{quotations: quotations, logger: logger}
}

type quotationResponse struct {
	ID                   string                  `json:"id"`
	ProcurementRequestID string                  `json:"procurement_request_id"`
	VendorName           string                  `json:"vendor_name"`
	OriginalFilename     string                  `json:"original_filename"`
	Status               domain.QuotationStatus  `json:"status"`
	CreatedAt            string                  `json:"created_at"`
}

type quotationsResponse struct {
	Quotations []quotationResponse `json:"quotations"`
}

// Upload accepts multipart/form-data with a "file" part plus a
// "vendor_name" field, and enqueues async extraction (202).
func (h *QuotationUploadHandler) Upload(c *gin.Context) {
	authCtx, ok := authOrReject(c)
	if !ok {
		return
	}

	// Request-size hint; the service enforces the real cap on bytes.
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		WriteError(c, http.StatusBadRequest, "bad_request",
			"A multipart/form-data file part named \"file\" is required.")
		return
	}
	defer file.Close()

	vendorName := c.PostForm("vendor_name")

	quotation, err := h.quotations.UploadQuotation(c.Request.Context(), authCtx, service.UploadQuotationInput{
		ProcurementRequestID: c.Param("id"),
		VendorName:           vendorName,
		OriginalFilename:    header.Filename,
		Content:             file,
	})
	if err != nil {
		var validation *service.ValidationError
		switch {
		case errors.As(err, &validation):
			WriteError(c, http.StatusUnprocessableEntity, "validation_error", validation.Error())
		case errors.Is(err, domain.ErrNotFound):
			WriteError(c, http.StatusNotFound, "not_found", "The requested resource was not found.")
		default:
			writeUnexpectedError(h.logger, c, "upload quotation", err)
		}
		return
	}

	c.JSON(http.StatusAccepted, toQuotationResponse(quotation))
}

// List returns a request's quotations, newest first.
func (h *QuotationUploadHandler) List(c *gin.Context) {
	if _, ok := authOrReject(c); !ok {
		return
	}

	quotations, err := h.quotations.QuotationsByRequest(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeUnexpectedError(h.logger, c, "list quotations", err)
		return
	}

	responses := make([]quotationResponse, 0, len(quotations))
	for _, quotation := range quotations {
		responses = append(responses, toQuotationResponse(quotation))
	}
	c.JSON(http.StatusOK, quotationsResponse{Quotations: responses})
}

func toQuotationResponse(quotation domain.Quotation) quotationResponse {
	return quotationResponse{
		ID:                   quotation.ID,
		ProcurementRequestID: quotation.ProcurementRequestID,
		VendorName:           quotation.VendorName,
		OriginalFilename:     quotation.OriginalFilename,
		Status:               quotation.Status,
		CreatedAt:            quotation.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}