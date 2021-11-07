
# Development setup

MacOS: `brew install gdal`
Linux: `sudo apt install libgdal-dev`

# Execution

    go run cmd/


# Obtaining input data

Downloaded data from https://environment.data.gov.uk/DefraDataDownload/?Mode=survey:
* Zoom to preferred area (you should see the tiles appear)
* Draw a square around it
* Click "Get Available Tiles"
* Choose preferred model and resolution and download tiles
* Currently testing with DTM

GeoTIFF format. Spec: http://geotiff.maptools.org/spec/contents.html

