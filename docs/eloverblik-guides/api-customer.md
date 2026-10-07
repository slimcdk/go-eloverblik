<!-- Source: https://docs.eloverblik.dk/docs/guides/api/customer, retrieved 2026-10-07 and converted from the rendered page to Markdown. Energinet's material, not covered by this repository's license; see docs/README.md. -->

# Customer API

Vil du gerne hente dine egne eldata via API? Følg denne trin-for-trin guide. Du behøver ikke være ekspert – vi tager det helt fra begyndelsen.

## Del 1: Opret din Refresh Token

1.  Log ind på [ElOverblik](https://eloverblik.dk) med MitID.
2.  Gå til menuen og vælg “API-adgang”.
3.  Klik på “Opret token” (kaldes også Refresh Token).
4.  Tillad brug af CPR-nummer (dette gør du kun én gang – fremtidige API-adgange kræver ikke dette trin).
5.  Navngiv token (fx “Hjem”).
6.  Kopiér tokenet – dette trin er vigtigt, da du skal bruge det senere.

✅ Nu har du din Refresh Token klar!

> **Advarsel**
>
> API nøgler skal holdes hemmelige, da alle med en nøgle kan få adgang til dine data. Pas på med hvordan du deler dine API nøgler, overvej at bruge sikker mail.

## Del 2: Få adgang til dine data

For at hente data skal du bruge en Data Access Token, som er gyldig i 24 timer.

Sådan gør du:

1.  Gå til [ElOverblik API Documentation](https://docs.eloverblik.dk/docs/api/customer).
2.  Indsæt din Refresh Token ved Bearer Token: og tryk Enter.
3.  Find afsnittet “Token”, klik på Test Request.
4.  Tryk derefter på Send.
5.  Din respons er din Data Access Token – kopier den.
6.  Gå tilbage til introduktionssiden og indsæt tokenet i authentication-feltet.

✅ Nu kan du hente dine data!

## Del 3: Hent dine elmålere

1.  Gå til afsnittet “Get Metering Points”.
2.  Kopiér det eller de elmåler-ID’er, du har brug for.

✅ Nu kan du se de elmålere, du ejer.

## Del 4: Hent dine måledata

1.  Brug dine elmåler-ID’er i afsnittet “MeterData”.
2.  Nu kan du hente data.

✅ Sådan, nu kan du se dit elforbrug!

God arbejdslyst!
