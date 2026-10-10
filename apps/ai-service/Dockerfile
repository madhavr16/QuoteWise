# syntax=docker/dockerfile:1

# Exact patch pin (ADR-011). The Go API and any future tooling must
# match this runtime: Python 3.12.
FROM python:3.12.15-slim-bookworm

# Local dev image: includes dev/test requirements so tests run without
# a separate toolchain on machines lacking Python 3.12 (risk R1).
# A leaner production image is a Milestone 7 concern.
RUN useradd --create-home --uid 10001 quotewise

WORKDIR /srv

COPY requirements.txt requirements-dev.txt ./
RUN pip install --no-cache-dir --root-user-action=ignore -r requirements-dev.txt

COPY app ./app

USER quotewise
EXPOSE 8000

ENTRYPOINT ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=6 \
  CMD python -c "import sys, urllib.request; sys.exit(0 if urllib.request.urlopen('http://localhost:8000/healthz', timeout=2).status == 200 else 1)"
