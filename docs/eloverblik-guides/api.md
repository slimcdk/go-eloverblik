<!-- Source: https://docs.eloverblik.dk/docs/guides/api, retrieved 2026-10-07 and converted from the rendered page to Markdown. Energinet's material, not covered by this repository's license; see docs/README.md. -->

# Brug af API

Med ElOverblik API, får du adgang til data på en fleksibel og automatiseret måde.

For at kunne hente eldata via Eloverblik API, skal du benytte en unik digital nøgle. Det gælder både, hvis du skal hente dine egne data, eller hvis du som tredjepart skal hente data på vegne af en bruger.

Den digitale nøgle giver adgang til dine eldata i op til 1 år. Herefter slettes den af hensyn til sikkerheden.

Du kan til enhver tid stoppe adgang til data ved at slette den digitale nøgle. Hvis du ønsker at forlænge adgangen til mere end 1 år, skal du danne en ny nøgle via ElOverblik.dk.

> **Tip**
>
> Start med at gå til en API client – dette kan f.eks være Postman. Du kan også benytte værktøjet på ElOverbliks [Customer API](https://docs.eloverblik.dk/docs/api/customer) eller [Thirdparty API](https://docs.eloverblik.dk/docs/api/thirdparty).
>
> Du kan læse nedenstående brugervejledninger for en trin-for-trin guide til brug af API.

## API'er

- [Customer API](https://docs.eloverblik.dk/docs/guides/api/customer): Denne API er designet til private -og erhvervskunder, der ønsker at hente egne data direkte fra ElOverblik.dk. Du får vejledning i opsætning, autentificering og brug af API’et.
- [Thirdparty API](https://docs.eloverblik.dk/docs/guides/api/thirdparty): Med ElOverblik.dk’s tredjeparts-API kan eksterne udbydere få adgang til kundedata – naturligvis med kundens samtykke.

## Generelt om API

**Hvad er et API?**

Et API står for Application Programming Interface. Det er en slags “bro” mellem forskellige systemer eller programmer, der gør det muligt for dem at kommunikere med hinanden.

Her er en enkel forklaring:

- Formål: Et API giver adgang til funktioner eller data fra et system uden at du behøver at kende hele systemets indre opbygning.
- Hvordan: Du sender en forespørgsel (request) til API’et, og det svarer med data eller udfører en handling (response).
- Eksempel: Når du bruger en app til at se vejrudsigten, henter den data fra en vejrserver via et API.

> **Refresh token**
>
> - En type Bearer token, der bruges til at verificere din identitet som bruger.
> - Anvendes til at generere et Data access token, som giver adgang til måledata (dette sker i et senere trin).

### Hvorfor bruge Eloverblik API?

Eloverblik tilbyder både en [webportal](https://eloverblik.dk/) (GUI) og et [API](https://docs.eloverblik.dk/docs/api/thirdparty). API’et er relevant, når du har brug for automatiseret adgang til data – enten som privat – og erhvervskunde eller som tredjepart.

#### Som privat- og erhvervskunde

- Automatisering: Hent dine egne forbrugsdata direkte til regneark, scripts eller dashboards uden manuelle eksportfiler.
- Stabil adgang: Brug et refresh token til at forny adgangstokens automatisk, så du slipper for gentagne logins.
- Performance: API’et er designet til høj belastning og egner sig til daglige opdateringer og analyser. Læs mere om vores [API begrænsninger.](https://docs.eloverblik.dk/docs/api/customer)

> **Eksempler**
>
> - Et hjemmedashboard, der viser gårsdagens timeforbrug og elpriser.
> - Scripts til at sammenligne forbrug over tid eller optimere efter grønne timer.

#### Som tredjepart

- Skalerbarhed: API’et understøtter millioner af kald pr. dag og bruges af mange aktive tredjeparter. Læs mere om vores [API begrænsninger.](https://docs.eloverblik.dk/docs/api/thirdparty)
- Kontrolleret datadeling: Adgang sker via kundens fuldmagt i Eloverblik, hvilket sikrer overholdelse af GDPR.
- Produktdifferentiering: Kombinér Eloverblik-data med egne data og algoritmer for at tilbyde personaliserede energiråd og automatisering.

> **Eksempler**
>
> - Apps til forbrugsanalyse og besparelsesråd.
> - Energioptimering for erhverv med mange elmålere.

## Sikkerhed og rettigheder

- Privatkunde: Opret et refresh token under Datadeling og hent data til egne systemer. Token kan deaktiveres når som helst.
- Tredjepart: Skal bruge et refresh token på samme måde som beskrevet ovenfor. Får kun adgang via kundens fuldmagt. Kunden kan til enhver tid trække samtykket tilbage.
