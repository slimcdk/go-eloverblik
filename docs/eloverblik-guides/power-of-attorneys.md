<!-- Source: https://docs.eloverblik.dk/docs/guides/power-of-attorneys, retrieved 2026-10-07 and converted from the rendered page to Markdown. Energinet's material, not covered by this repository's license; see docs/README.md. -->

# Fuldmagter

En fuldmagt er en aftale, som skabes mellem en godkendt tredjepartsvirksomhed og en privat elforbruger eller en erhvervs-elforbruger. Med fuldmagten kan tredjepartsvirksomheden hente eldata fra DataHub-systemet via ElOverblik på vegne af elforbrugeren.

> **Info**
>
> Fuldmagter er nødvendige, når en tredjepart skal have adgang til kundedata.

Lær hvordan du opretter en fuldmagt som tredjepart og hvordan du som bruger godkender en fuldmagt:

### Opret en fuldmagt

1.  Log på [ElOverblik](https://eloverblik.dk) via tredjepartsportalen med MitID Erhverv.
2.  Du kan via portalen oprette et personligt fuldmagtslink ved at bruge funktionen ’Opret link’.

#### Disse felter skal udfyldes

- Dato (Obligatorisk): Datofelterne er forhåndsudfyldte, men du har mulighed for at ændre dem.
  - Fra dato: maksimalt løbende år + 5 år tilbage.
  - Til dato: minimum 3 måneder eller maksimalt 3 år frem i tiden.
- Kundenøgle: Kundenøgle bruges til at skelne mellem fuldmagter fra forskellige kunder.
- Returnerings URL-adresse: Hvis du ønsker at få videresendt elforbrugerne til en bestemt hjemmeside, efter at fuldmagten er givet.
- Du sender fuldmagtslinket til elforbrugeren.

#### Nu sker dette hos elforbrugeren

1.  Elforbrugeren klikker på linket og bliver bedt om at logge ind på ElOverblik ved hjælp af MitID.
2.  Elforbrugeren bliver præsenteret for fuldmagten og har mulighed for at vælge i hvor lang tid, fuldmagtsaftalen skal gælde.
3.  Fuldmagten accepteres og herefter bliver der automatisk oprettet en relation i DataHub-systemet, hvor tredjeparten registreres på elmåleren.
4.  Elforbrugeren kan til enhver tid tilbagekalde fuldmagten.

Nu har din virksomhed adgang til elforbrugerens måledata via tredjepartsportalen eller via tredjeparts-API.

### Tredjeparts-API

Tredjepart-API er udviklet som et standard REST-API og bruges af tredjepartsvirksomheden, som har fuldmagt til at hente måledata på vegne af elforbrugeren.

Fordelen ved at bruge tredjepart-API i forhold til kunde-API er, at fuldmagten er gyldig i helt op til 3 år. Der er også færre tekniske begrænsninger på tredjeparts-API’et.

Du kan gå direkte videre til at benytte API’en [her.](https://docs.eloverblik.dk/docs/api/thirdparty)
