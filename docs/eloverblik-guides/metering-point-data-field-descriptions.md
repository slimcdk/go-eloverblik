<!-- Source: https://docs.eloverblik.dk/docs/guides/metering-point-data-field-descriptions, retrieved 2026-10-07 and converted from the rendered page to Markdown. Energinet's material, not covered by this repository's license; see docs/README.md. -->

# Feltbeskrivelser for elmålerdata

Når du downloader stamdata for dine elmålere (kaldet målepunkter i eksporten og API’et) - eller tilgår data via fuldmagt eller datadeling – deles en række felter med oplysninger om din elmåler, tilslutning, adresse og leverandørforhold.

Denne side forklarer, hvad de enkelte felter betyder, så du nemt kan forstå indholdet i dine data. Felterne er grupperet efter emne for overblikkets skyld.

> **Bemærk:** Ikke alle felter er synlige i alle sammenhænge. Hvilke felter, du kan se, afhænger af om du tilgår data som kunde, via datadeling eller via fuldmagt til tredjepart.

> Elleverandørstartdato, Elvarmestartdato og Kunde_start_dato er p.t. utilgængelige.

------------------------------------------------------------------------

## Identifikation af elmåler

| Felt                            | Beskrivelse                                                                     |
|---------------------------------|---------------------------------------------------------------------------------|
| **MålepunktsID**            | Det unikke 18-cifrede ID (GSRN-nummer), der identificerer din elmåler i DataHub |
| **MålepunktsID_hovedmåler** | ID på den overordnede hovedmåler, hvis din elmåler er en bimåler                |
| **Alias**                   | Et valgfrit kaldenavn, du selv kan give elmåleren                               |

------------------------------------------------------------------------

## Type og klassifikation

| Felt                   | Beskrivelse                                                                                                                  |
|------------------------|------------------------------------------------------------------------------------------------------------------------------|
| **Målepunktstype** | Angiver om elmåleren er af typen forbrug, produktion eller udveksling m.m.                                                   |
| **Målepunktsart**  | Angiver, om elmåleren er fysisk, virtuel eller beregnet                                                                      |
| **Energi_type**    | Typen af energi                                                                                                              |
| **Produkt**        | Angiver, hvad måleren måler. For de fleste er det "Aktiv energi", altså den strøm du rent faktisk forbruger eller producerer |
| **Måleenhed**      | Enhedstype (typisk kWh)                                                                                                      |

------------------------------------------------------------------------

## Tilslutning og kapacitet

| Felt                        | Beskrivelse                                                                            |
|-----------------------------|----------------------------------------------------------------------------------------|
| **Tilslutningsstatus**  | Om elmåleren er nyoprettet, tilsluttet eller afbrudt                                   |
| **Tilslutningstype**    | Hvordan installationen er tilsluttet elnettet (f.eks. direkte eller via transformator) |
| **Effektgrænse_kW**     | Den maksimale effekt i kilowatt (kW) elmåleren er godkendt til                         |
| **Effektgrænse_ampere** | Den maksimale effekt i ampere (A) elmåleren er godkendt til                            |
| **Anlægskapacitet**     | Den installerede kapacitet for produktionsanlæg (relevant for solceller m.v.)          |
| **Afbrydelsesart**      | Hvordan forsyningen kan afbrydes (manuelt eller fjernstyret)                           |

------------------------------------------------------------------------

## Netområde og afregning

| Felt                            | Beskrivelse                                                                 |
|---------------------------------|-----------------------------------------------------------------------------|
| **Netområde**               | Det geografiske netområde, som elmåleren tilhører                           |
| **Nettoafregningsgruppe**   | Gruppe for nettoafregning (relevant for producenter med f.eks. solceller)   |
| **Afregningsform** (udgået) | Hvordan elforbruget afregnes (f.eks. timeafregning eller skabelonafregning) |
| **Aftagepligt**             | Om der er pligt til at aftage el via elmåleren                              |
| **Branchekode** (udgået)    | Branchekode, der klassificerer typen af forbrug (privat, erhverv m.v.)      |

------------------------------------------------------------------------

## Aflæsning og måler

*Disse felter beskriver den fysiske elmåler.*

| Felt                                   | Beskrivelse                                                                 |
|----------------------------------------|-----------------------------------------------------------------------------|
| **Målernummer**                    | Serienummeret på den fysiske elmåler                                        |
| **Målertype** (udgået)             | Typen af måler (f.eks. fjernaflæst smart-måler)                             |
| **Målercifre** (udgået)            | Antal cifre på målerens tælleværk                                           |
| **Måleromregningsfaktor** (udgået) | Faktor, der ganges på målerens aflæsning for at omregne til faktisk forbrug |
| **Målerenhed** (udgået)            | Enheden måleren registrerer i                                               |
| **Aflæsningsfrekvens** (udgået)    | Hvor ofte måleren aflæses (f.eks. time, dag, måned)                         |
| **Anslået_årsforbrug** (udgået)    | Det estimerede årlige elforbrug i kWh                                       |

------------------------------------------------------------------------

## Adresse (elmålerens lokation)

*Disse felter angiver den fysiske adresse hvor elmåleren er installeret.*

| Felt                         | Beskrivelse                                                          |
|------------------------------|----------------------------------------------------------------------|
| **Adressekode** (udgået) | Unik adressekode fra Danmarks Adresseregister (DAR)                  |
| **Vejnavn**              | Vejnavn                                                              |
| **Husnummer**            | Husnummer                                                            |
| **Etage**                | Etage                                                                |
| **Dørnummer**            | Dørnummer (lejlighed/side)                                           |
| **Postnummer**           | Postnummer                                                           |
| **By**                   | Bynavn                                                               |
| **Stednavn**             | Eventuelt stednavn/lokalitetsnavn                                    |
| **Kommunekode**          | Kommunens administrative kode                                        |
| **DAR_adresse_konflikt** | Angiver om der er uoverensstemmelse mellem elmålerens adresse og DAR |
| **DAR_reference**        | Reference-ID til den officielle adresse i Danmarks Adresseregister   |

------------------------------------------------------------------------

## Elleverandør og netvirksomhed

> **Bemærk:** Elleverandøroplysninger er synlige for kunden selv og ved almindelig datadeling, men deles **ikke** med tredjeparter via fuldmagt.

| Felt                                                     | Beskrivelse                                                   |
|----------------------------------------------------------|---------------------------------------------------------------|
| **Elleverandør**                                     | Navnet på din elleverandør (elhandelsselskab)                 |
| **Elleverandør_Id** / **Elleverandør_Id_type**   | Elleverandørens unikke identifikationsnummer og type          |
| **Elleverandørstartdato** (utilgængelig)             | Dato for hvornår den nuværende elleverandør overtog elmåleren |
| **Netvirksomhed**                                    | Navnet på den netvirksomhed, der ejer elnettet i dit område   |
| **Netvirksomhed_Id** / **Netvirksomhed_Id_type** | Netvirksomhedens unikke identifikationsnummer og type         |

------------------------------------------------------------------------

## Kundeforhold

| Felt                                    | Beskrivelse                                               |
|-----------------------------------------|-----------------------------------------------------------|
| **Kunde_start_dato** (utilgængelig) | Dato for hvornår det nuværende kundeforhold startede      |
| **CVR-nummer**                      | CVR-nummer på den virksomhed, der er tilknyttet elmåleren |
| **Målepunktskommentar**             | Fritekstkommentar tilknyttet elmåleren                    |

------------------------------------------------------------------------

## Elvarme og afgifter

| Felt                                    | Beskrivelse                                                      |
|-----------------------------------------|------------------------------------------------------------------|
| **Reduceret_elafgift**              | Om elmåleren er godkendt til reduceret elafgift (f.eks. elvarme) |
| **Elvarmestartdato** (utilgængelig) | Dato for hvornår elvarmegodkendelsen trådte i kraft              |

------------------------------------------------------------------------

## Teknisk kontakt

*Kontaktoplysninger for den teknisk ansvarlige for installationen. Alle felter med præfiks `Teknisk_kontakt_` er grupperet her.*

| Felt                                                                                                                                                                                                                                                           | Beskrivelse                                      |
|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------|
| **Teknisk_kontakt_Navn** / **Teknisk_kontakt_Navn2**                                                                                                                                                                                                   | Navn (og evt. supplerende navnelinje)            |
| **Teknisk_kontakt_Vejnavn** , **Teknisk_kontakt_Husnr.** , **Teknisk_kontakt_Etage** , **Teknisk_kontakt_Dør** , **Teknisk_kontakt_By** , **Teknisk_kontakt_Postnr** , **Teknisk_kontakt_Stednavn** , **Teknisk_kontakt_Land** | Adresseoplysninger                               |
| **Teknisk_kontakt_Telefonnr.** / **Teknisk_kontakt_Mobilnr.**                                                                                                                                                                                          | Telefon- og mobilnummer                          |
| **Teknisk_kontakt_E-mail**                                                                                                                                                                                                                                 | E-mailadresse                                    |
| **Teknisk_kontakt_Attention**                                                                                                                                                                                                                              | Attention-linje (modtager)                       |
| **Teknisk_kontakt_Postbox**                                                                                                                                                                                                                                | Postboksnummer                                   |
| **Teknisk_kontakt_beskyttet_adresse**                                                                                                                                                                                                                      | Om kontaktens adresse er navne-/adressebeskyttet |

------------------------------------------------------------------------

## Juridisk kontakt

*Kontaktoplysninger for den juridisk ansvarlige (ejer/lejer). Alle felter med præfiks `Juridisk_kontakt_` er grupperet her — strukturen er identisk med teknisk kontakt.*

| Felt                                                                                                                                                                                                                                                                    | Beskrivelse                                      |
|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------|
| **Juridisk_kontakt_Navn** / **Juridisk_kontakt_Navn2**                                                                                                                                                                                                          | Navn (og evt. supplerende navnelinje)            |
| **Juridisk_kontakt_Vejnavn** , **Juridisk_kontakt_Husnr.** , **Juridisk_kontakt_Etage** , **Juridisk_kontakt_Dør** , **Juridisk_kontakt_By** , **Juridisk_kontakt_Postnr.** , **Juridisk_kontakt_Stednavn** , **Juridisk_kontakt_Land** | Adresseoplysninger                               |
| **Juridisk_kontakt_Telefonnr.** / **Juridisk_kontakt_Mobilnr.**                                                                                                                                                                                                 | Telefon- og mobilnummer                          |
| **Juridisk_kontakt_E-mail**                                                                                                                                                                                                                                         | E-mailadresse                                    |
| **Juridisk_kontakt_Attention**                                                                                                                                                                                                                                      | Attention-linje (modtager)                       |
| **Juridisk_kontakt_Postbox**                                                                                                                                                                                                                                        | Postboksnummer                                   |
| **Juridisk_kontakt_beskyttet_adresse**                                                                                                                                                                                                                              | Om kontaktens adresse er navne-/adressebeskyttet |

------------------------------------------------------------------------

## Navnebeskyttelse

| Felt                   | Beskrivelse                                                   |
|------------------------|---------------------------------------------------------------|
| **Beskyttet_Navn** | Angiver om kundens navn er beskyttet (navnebeskyttelse i CPR) |

------------------------------------------------------------------------

## Priser

*Disse felter beskriver de tariffer og afgifter, der er knyttet til elmåleren.*

| Felt                       | Beskrivelse                                                               |
|----------------------------|---------------------------------------------------------------------------|
| **MålepunktsID**       | Den elmåler (GSRN-nummer), som prisen er tilknyttet                       |
| **Pristype**           | Typen af pris (f.eks. abonnement, tarif eller afgift)                     |
| **Pris_ID**            | Unikt ID for den pågældende priskomponent                                 |
| **Navn**               | Navnet på prisen/tariffen (f.eks. "Nettarif C time" eller "Systemtarif")  |
| **Beskrivelse**        | Uddybende beskrivelse af hvad prisen dækker                               |
| **Ejer**               | Hvem der opkræver prisen (f.eks. netvirksomhed, Energinet eller staten)   |
| **Gyldig_fra**         | Dato for hvornår prisen træder i kraft                                    |
| **Gyldig_til**         | Dato for hvornår prisen udløber                                           |
| **Position**           | Tidsposition inden for døgnet (relevant for tidsdifferentierede tariffer) |
| **Pris (Ekskl. Moms)** | Prisbeløbet eksklusiv moms                                                |
| **Mængde**             | Den mængde prisen gælder for (typisk pr. kWh eller pr. dag)               |

------------------------------------------------------------------------
