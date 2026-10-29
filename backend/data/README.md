# Tamil Nadu attraction data

`tamil_nadu_attractions.json` contains 791 named tourism and historic features fetched from OpenStreetMap with the Overpass API on 2026-10-22. Visitor detail tags were refreshed on 2026-10-29 from an Overpass snapshot whose OSM data timestamp was 2026-07-24T11:04:51Z. The import includes named attractions, museums, viewpoints, zoos, theme parks, aquariums, galleries, artwork, monuments, memorials, archaeological sites, forts, castles, and ruins within Tamil Nadu's administrative boundary. One duplicate name and coordinate was removed.

Each record retains its OpenStreetMap element URL. The data is available under the Open Database License (ODbL); see <https://www.openstreetmap.org/copyright>. The application displays OpenStreetMap attribution alongside imported place results.

The enrichment command fetches OSM tags such as opening hours, address, wheelchair access, contact details, fees, toilets, internet access, Wikipedia, and Wikidata. Coverage is incomplete because these details are only available when mapped by contributors. A missing field means the source has no mapped value; it does not confirm that a venue lacks the facility. Re-run enrichment and import to refresh the details:

From the `backend` directory, apply migrations and import or refresh the records with:

```sh
go run ./cmd/enrichosm
go run ./cmd/migrate up
go run ./cmd/importosm
```

Set `OVERPASS_URL` to use another compatible Overpass instance; see <https://wiki.openstreetmap.org/wiki/Overpass_API> for public instance information.
