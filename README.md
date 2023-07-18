# go-stac-server
A dynamic stac 1.0.0 server implemented in golang

# Requirements

1. PostgreSQL 13+
2. PostGIS 3+
3. [pgstac](https://github.com/stac-utils/pgstac) schema
4. go-stac-server
5. geospatial data

# Quickstart

```bash
# install pypgstac and initialize database
pip install pypgstac
pip install "pypgstac[psycopg]"
export DSN=postgresql://stac@localhost:5432/stac
pypgstac migrate
go-stac-server
```

In a web browser, navigate to `https://localhost:3000/` to browse the catalog.

# Configuration

| Command Flag | Environment Variable | Configuration File | Description                                                                                         |
|--------------|----------------------|--------------------|-----------------------------------------------------------------------------------------------------|
| --dsn        | DSN                  | database.dsn       | Database connection string `postgresql://[[username:[password]@][host[:port]][/dbname][?paramspec]` |
| --port       | PORT                 | server.port        | Port to run server on                                                                               |

## Sample configuration file:

```toml
[server]
port=3000

[database]
dsn="postgresql://stac@localhost:5432/stac"

[stac.catalog]
id="stac-catalog"
title="STAC API"
description="go-stac-server STAC API"
```

(PgSTAC)[https://stac-utils.github.io/pgstac/pgstac/] provides the backend database
and offers several configuration options. See their documentation for specifics on
what options are available. 

# Implemented STAC extensions

| Title                                                                    | Scope                     | Version        | Description                                                                                                                                                                                                         |
|--------------------------------------------------------------------------|---------------------------|----------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [Accuracy](https://github.com/stac-extensions/accuracy)                  | Collection, Item          | **Unreleased** | Fields to provide estimates of accuracy, both geometric and measurement (e.g., radiometric) accuracy.                                                                                                               |
| [Alternate Assets](https://github.com/stac-extensions/alternate-assets)  | Asset                     | 1.1.0          | Describes alternate locations and mirrors of assets                                                                                                                                                                 |
| [Classification](https://github.com/stac-extensions/classification)      | Collection, Item          | 1.1.0          | Describes categorical values and bitfields to give values in a file a certain meaning (classification).                                                                                                             |
| [Composite](https://github.com/stac-extensions/composite)                | Item                      | **Unreleased** | Defines how virtual assets can be composed from existing assets in STAC                                                                                                                                             |
| [Electro-Optical](https://github.com/stac-extensions/eo)                 | Collection, Item          | 1.1.0          | Covers electro-optical data that represents a snapshot of the Earth. It could consist of cloud cover and multiple spectral bands, for example visible bands, infrared bands, red edge bands and panchromatic bands. |
| [Example Links](https://github.com/stac-extensions/example-links)        | Catalog, Collection, Item | **Unreleased** | Allows to provide links to examples, e.g. code snippets.                                                                                                                                                            |
| [File Info](https://github.com/stac-extensions/file)                     | Catalog, Collection, Item | 2.1.0          | Specifies file-related details such as size, data type and checksum for assets and links in STAC.                                                                                                                   |
| [Item Assets Definition](https://github.com/stac-extensions/item-assets) | Collection                | 1.0.0          | Provides a way to specify details about what assets may be found in Items belonging to a Collection.                                                                                                                |
| [Label](https://github.com/stac-extensions/label)                        | Collection, Item          | 1.0.1          | Items that relate labeled AOIs with source imagery.                                                                                                                                                                 |
| [Processing](https://github.com/stac-extensions/processing)              | Collection, Item          | 1.1.0          | Indicates from which processing chain data originates and how the data itself has been produced.                                                                                                                    |
| [Projection](https://github.com/stac-extensions/projection)              | Collection, Item          | 1.1.0          | Provides a way to describe Items whose assets are in a geospatial projection.                                                                                                                                       |
| [Stats](https://github.com/stac-extensions/stats)                        | Catalog, Collection       | 0.2.0          | Describes the number of items, extensions and assets that are contained in a STAC catalog.                                                                                                                          |
| [Timestamps](https://github.com/stac-extensions/timestamps)              | Catalog, Collection, Item | 1.1.0          | Allows to specify numerous additional timestamps for assets and metadata.                                                                                                                                           |
| [Versioning Indicators](https://github.com/stac-extensions/version)      | Collection, Item          | 1.2.0          | Provides fields and link relation types to provide a version and indicate deprecation.                                                                                                                              |
| [View Geometry](https://github.com/stac-extensions/view)                 | Collection, Item          | 1.0.0          | View Geometry adds metadata related to angles of sensors and other radiance angles that affect the view of resulting data.                                                                                          |

# Errors

go-stac-server logs most errors using structured logging. For fatal errors the
application will exit with a non-zero exit code.

| Exit Code | Description                     |
|-----------|---------------------------------|
| 0         | Application exited successfully |
| 66        | Could not connect to database   |
| 73        | Could not bind to server port   |