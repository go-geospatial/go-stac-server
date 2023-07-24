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

[PgSTAC](https://stac-utils.github.io/pgstac/pgstac/) provides the backend database
and offers several configuration options. See their documentation for specifics on
what options are available.

| Title                                                             | Version    | Description                                                                                                                    |
|-------------------------------------------------------------------|------------|--------------------------------------------------------------------------------------------------------------------------------|
| [Browseable](https://github.com/stac-api-extensions/browseable)   | 1.0.0-rc.3 | Browseable advertises all Items in a STAC API Catalog can be reached by traversing child and item links.                       |
| [Context](https://github.com/stac-api-extensions/context)         | 1.0.0-rc.2 | Context Extension                                                                                                              |
| [Fields](https://github.com/stac-api-extensions/fields)           | 1.0.0-rc.3 | The Fields Extensions describes a mechanism to include or exclude certain fields from a response.                              |
| [Filter](https://github.com/stac-api-extensions/filter)           | 1.0.0-rc.2 | The Filter extension provides an expressive mechanism for searching based on Item attributes.                                  |
| [Query](https://github.com/stac-api-extensions/query)             | 1.0.0-rc.2 | The Query Extension adds a query parameter that allows additional filtering based on the properties of Item objects.           |
| [Sort](https://github.com/stac-api-extensions/sort)               | 1.0.0-rc.2 | The Sort Extension that allows the user to define the fields by which to sort results.                                         |
| [Transaction](https://github.com/stac-api-extensions/transaction) | 1.0.0-rc.2 | The Transaction Extension supports the creation, editing, and deleting of items through POST, PUT, PATCH, and DELETE requests. |

# Errors

go-stac-server logs most errors using structured logging. For fatal errors the
application will exit with a non-zero exit code.

| Exit Code | Description                     |
|-----------|---------------------------------|
| 0         | Application exited successfully |
| 66        | Could not connect to database   |
| 73        | Could not bind to server port   |