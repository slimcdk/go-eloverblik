<!-- Source: https://docs.eloverblik.dk/docs/guides/api/thirdparty, retrieved 2026-10-07 and converted from the rendered page to Markdown. Energinet's material, not covered by this repository's license; see docs/README.md. -->

# Thirdparty API

> **Adgang til Tredjeparts API**
>
> For at benytte tredjeparts-API’et skal du:
>
> 1.  Være godkendt tredjepart.
> 2.  Have indhentet fuldmagt fra dataejeren.

## Del 1: Opret din Refresh Token

1.  Log ind på [ElOverblik](https://eloverblik.dk) med MitID Erhverv.
2.  Vælg “API-adgang” i menuen.
3.  Klik på “Opret refresh token” (gyldig i 1 år).
4.  Tokenet er af typen Bearer Token, som bruges til at bekræfte din identitet som tredjepart.
5.  Kopiér token – du skal bruge det senere.

✅ Nu har du din Refresh Token klar!

> **Advarsel**
>
> API nøgler skal holdes hemmelige, da alle med en nøgle kan få adgang til dine data. Pas på med hvordan du deler dine API nøgler, overvej at bruge sikker mail.

## Del 2: Få adgang til måledata

For at hente data skal du bruge en Data Access Token, som er gyldig i 24 timer.

Sådan gør du:

1.  Gå til [ElOverblik API Documentation](https://docs.eloverblik.dk/docs/api/thirdparty).
2.  Indsæt din Refresh Token under Bearer Token.
3.  Brug endpointet “Get data access token” med din Refresh Token som input.
4.  Responsen er din Data Access Token – kopier den.

✅ Nu kan du hente data!

## Del 3: Få en liste over fuldmagter

1.  Brug endpointet “GetAuthorizations” med din Data Access Token.
2.  Du får en liste over tilgængelige fuldmagter.

## Del 4: Hent elmålere

1.  Brug endpointet “GetMeteringPoints” med parametre som:
    - Fuldmagts-ID
    - CVR-nummer (virksomhed)
    - Kundenøgle (private)
2.  Du får en liste med elmåler-ID’er.

## Del 5: Hent måledata

Med elmåler-ID’er kan du bruge følgende endpoints:

- GetMeteringPointDetails – stamdata for elmåler.
- GetTimeSeries – forbrug og produktion som tidsserie.

God arbejdslyst!
