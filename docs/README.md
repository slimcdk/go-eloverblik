# Energinet's API documentation

The files in this directory are Energinet's documentation of the Eloverblik Customer and
Third-Party APIs, kept here so the client can be checked against them offline. They are
Energinet's material, copied as published, and are **not** covered by this repository's MIT
license.

| File | What it is | Source | Retrieved |
|------|------------|--------|-----------|
| `swagger-eloverblik-customerapi.json` | OpenAPI 3.0.4 document of the Customer API, `api-version` 1.0, byte for byte as served | <https://api.eloverblik.dk/customerapi/swagger/customerapi-v1.0/swagger.json> | 2026-10-07, unchanged since 2026-10-05 |
| `swagger-eloverblik-thirdpartyapi.json` | OpenAPI 3.0.4 document of the Third-Party API, `api-version` 1.0, byte for byte as served | <https://api.eloverblik.dk/thirdpartyapi/swagger/thirdpartyapi-v1.0/swagger.json> | 2026-10-07, unchanged since 2026-10-05 |
| `customer-and-third-party-api-for-datahub-eloverblik-technical-description.pdf` | Technical description of both APIs, document 19/11830-1, last revised 27 March 2025 | <https://energinet.dk/media/2l1lmb2z/customer-and-third-party-api-for-datahub-eloverblik-technical-description.pdf> | 2026-10-07 |
| `eloverblik-guides/*.md` | The guides on docs.eloverblik.dk, in Danish, converted from the rendered pages to Markdown; each file names its page | <https://docs.eloverblik.dk/docs/guides/introduction> | 2026-10-07 |
| `MyEnergyDataMarketDocumentResponse.json`, `metering-point-details-response.json`, `metering-point-price-data.json`, `meter-point-readings-response.json` | Example response skeletons from the 2022 documentation, with `"string"` placeholders. Superseded by the OpenAPI documents; the meter readings endpoint no longer exists | Energinet's API documentation of 2022 | 2022-02 |

## Which one to trust

The OpenAPI documents are the most current: they describe the API as it is after DataHub
3.0, which went into operation on 18 September 2026. The technical description predates
DataHub 3.0. It still describes linking a metering point with a web access code and
deleting a relation, which both answer `410 Gone` since then. Where the two disagree, the
OpenAPI documents win; where those disagree with the live API, the live API wins, as with
the time series resolutions described in `llms.md`.

The guides describe the portal and the data, not the endpoints. The field descriptions are
the one place that lists the master data fields Energinet has retired, or made unavailable
for now, since DataHub 3.0.

## Refreshing

Fetch the documents by hand, once, and diff them against the copies here:

```bash
for api in customerapi thirdpartyapi; do
  curl -sS -o "docs/swagger-eloverblik-$api.json" \
    "https://api.eloverblik.dk/$api/swagger/$api-v1.0/swagger.json"
done
git diff --stat docs/
```

The project deliberately runs no scheduled check against Energinet's servers.

docs.eloverblik.dk renders its pages in the browser, so a plain download returns only an
empty page shell. The guides were rendered with a headless browser that waits for the page
to render (`google-chrome --headless=new --virtual-time-budget=15000 --dump-dom <url>`), and
only the article body, the `div.theme-doc-markdown` element, was converted with pandoc
(`-f html -t gfm`).
