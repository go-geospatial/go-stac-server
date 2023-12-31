# Developing

## Update stac browser

(STAC Browser)[https://github.com/radiantearth/stac-browser] is embedded with
go-stac-server as a convenience to browse catalog data. A copy built with
NPM is stored in the `static/files` directory.

To update, perform the following steps:

1. Clone the repository from `git@github.com:radiantearth/stac-browser.git`
2. Apply stac-browser.patch and build:

```bash
git clone git@github.com:radiantearth/stac-browser.git
patch < ../stac-browser.patch
npm install
npm run build --historyMode=hash
```

1. Delete the contents of `static/files/css` and `static/files/js`
2. Copy the files from `dist` into `static/files`
3. Delete `*.map` files

NOTE: `config.js` is dynamically generated by the `handler.StacBrowserConfig` function and may be customized with
the --gui-config option

## Update Swagger UI

1. Download latest release of swagger standalone
2. Replace the files in `static/files/doc` with the files in the `dist` folder
3. Edit the file `swagger-initializer.js` and change the url to `/openapi.json`
4. Edit the file `swagger-initializer.js` and change the layout to `BaseLayout`
5. Rename `index.html` to `openapi.html`
