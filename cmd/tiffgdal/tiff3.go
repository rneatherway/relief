package main

import (
	"fmt"
	"os"

	"github.com/airbusgeo/godal"
	"github.com/hschendel/stl"
)

type PixelBuffer struct {
	buf    []float32
	width  int
	height int
}

// Imagine width is three, height is two and pixel data is:
//
// a b c
// d e f
//
// This will be in the buf as: a b c d e f
// So for each 'y' we need to advance by 'width'
// x + y*width
func (pb *PixelBuffer) get(x int, y int) float32 {
	return pb.buf[x+y*pb.width]
}

func (pb *PixelBuffer) getVec3(x int, y int) stl.Vec3 {
	return stl.Vec3{
		float32(x),
		float32(y),
		pb.get(x, y),
	}
}

func (pb *PixelBuffer) toSTL(x1 int, y1 int, x2 int, y2 int) stl.Solid {
	solid := stl.Solid{}

	// Top
	for i := x1; i < x2; i++ {
		for j := y1; j < y2; j++ {
			// TODO: I think we can factor out this 4 points -> 2 triangles thing
			tl := pb.getVec3(i, j)
			tr := pb.getVec3(i+1, j)
			bl := pb.getVec3(i, j+1)
			br := pb.getVec3(i+1, j+1)

			solid.AppendTriangle(stl.Triangle{Vertices: [3]stl.Vec3{tl, bl, tr}})
			solid.AppendTriangle(stl.Triangle{Vertices: [3]stl.Vec3{tr, bl, br}})
		}
	}

	for i := x1; i < x2; i++ {
		// Back
		tr := pb.getVec3(i, 0)
		tl := pb.getVec3(i+1, 0)
		br := stl.Vec3{float32(i), 0, 0}
		bl := stl.Vec3{float32(i + 1), 0, 0}

		solid.AppendTriangle(stl.Triangle{Vertices: [3]stl.Vec3{tl, bl, tr}})
		solid.AppendTriangle(stl.Triangle{Vertices: [3]stl.Vec3{tr, bl, br}})

		// Front
		tl = pb.getVec3(i, y2)
		tr = pb.getVec3(i+1, y2)
		bl = stl.Vec3{float32(i), float32(y2), 0}
		br = stl.Vec3{float32(i + 1), float32(y2), 0}

		solid.AppendTriangle(stl.Triangle{Vertices: [3]stl.Vec3{tl, bl, tr}})
		solid.AppendTriangle(stl.Triangle{Vertices: [3]stl.Vec3{tr, bl, br}})
	}

	solid.RecalculateNormals()
	solid.Validate()
	return solid
}

func FromGeoTIFF(path string) (*PixelBuffer, error) {
	godal.RegisterAll()
	hDataset, err := godal.Open(path)
	if err != nil {
		return nil, err
	}
	defer hDataset.Close()

	structure := hDataset.Structure()
	band := hDataset.Bands()[0]
	buf := make([]float32, structure.SizeX*structure.SizeY)
	err = band.Read(0, 0, buf, structure.SizeX, structure.SizeY)
	if err != nil {
		return nil, err
	}

	return &PixelBuffer{buf: buf, width: structure.SizeX, height: structure.SizeY}, nil
}

func realMain() error {
	pb, err := FromGeoTIFF("P_10719/DTM_SP5005_P_10719_20200315_20200315.tif")
	if err != nil {
		return err
	}

	_, err = fmt.Println(pb.get(0, 0))
	return err
}

func main() {
	err := realMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return
	/*
		GDALDatasetH  hDataset;
		GDALAllRegister();
		hDataset = GDALOpen( pszFilename, GA_ReadOnly );
		if( hDataset == NULL )
		{
			...;
		}
	*/
	godal.RegisterAll()
	hDataset, err := godal.Open("P_10719/DTM_SP5005_P_10719_20200315_20200315.tif")
	if err != nil {
		panic(err)
	}

	/*
		hDriver = GDALGetDatasetDriver( hDataset );
		printf( "Driver: %s/%s\n",
			GDALGetDriverShortName( hDriver ),
			GDALGetDriverLongName( hDriver ) );
	*/
	//not implemented

	/*
		printf( "Size is %dx%dx%d\n",
			GDALGetRasterXSize( hDataset ),
			GDALGetRasterYSize( hDataset ),
			GDALGetRasterCount( hDataset ) );
	*/
	structure := hDataset.Structure()
	fmt.Printf("Size is %dx%dx%d\n", structure.SizeX, structure.SizeY, structure.NBands)

	/*
		if( GDALGetProjectionRef( hDataset ) != NULL )
			printf( "Projection is '%s'\n", GDALGetProjectionRef( hDataset ) );
	*/
	if pj := hDataset.Projection(); pj != "" {
		fmt.Printf("Projection is '%s'\n", pj)
	}

	/*
		if( GDALGetGeoTransform( hDataset, adfGeoTransform ) == CE_None )
		{
			printf( "Origin = (%.6f,%.6f)\n",
				adfGeoTransform[0], adfGeoTransform[3] );
			printf( "Pixel Size = (%.6f,%.6f)\n",
				adfGeoTransform[1], adfGeoTransform[5] );
		}
	*/
	if gt, err := hDataset.GeoTransform(); err == nil {
		fmt.Printf("Origin = (%.6f,%.6f)\n", gt[0], gt[3])
		fmt.Printf("Pixel Size = (%.6f,%.6f)\n", gt[1], gt[5])
	}

	/*
		GDALRasterBandH hBand;
		int             nBlockXSize, nBlockYSize;
		int             bGotMin, bGotMax;
		double          adfMinMax[2];
		hBand = GDALGetRasterBand( hDataset, 1 );
		GDALGetBlockSize( hBand, &nBlockXSize, &nBlockYSize );
		printf( "Block=%dx%d Type=%s, ColorInterp=%s\n",
				nBlockXSize, nBlockYSize,
				GDALGetDataTypeName(GDALGetRasterDataType(hBand)),
				GDALGetColorInterpretationName(
					GDALGetRasterColorInterpretation(hBand)) );
	*/
	band := hDataset.Bands()[0] //Note that in godal, bands are indexed starting from 0, not 1
	bandStructure := band.Structure()
	fmt.Printf("Block=%dx%d Type=%s, ColorInterp=%s\n",
		bandStructure.BlockSizeX, bandStructure.BlockSizeY,
		bandStructure.DataType,
		band.ColorInterp().Name())

	buf := make([]float32, structure.SizeX*structure.SizeY)
	err = band.Read(0, 0, buf, structure.SizeX, structure.SizeY)
	if err != nil {
		panic(err)
	}

	solid := stl.Solid{}
	maxX := 10
	maxY := 10
	for i := 0; i < maxX-1; i++ {
		for j := 0; j < maxY-1; j++ {
			solid.AppendTriangle(
				stl.Triangle{
					Vertices: [3]stl.Vec3{
						{float32(i), float32(j), buf[i+j*structure.SizeX]},
						{float32(i + 1), float32(j), buf[i+1+j*structure.SizeX]},
						{float32(i), float32(j + 1), buf[i+(j+1)*structure.SizeX]},
					},
				},
			)
			solid.AppendTriangle(
				stl.Triangle{
					Vertices: [3]stl.Vec3{
						{float32(i + 1), float32(j), buf[i+1+j*structure.SizeX]},
						{float32(i + 1), float32(j + 1), buf[i+1+(j+1)*structure.SizeX]},
						{float32(i), float32(j + 1), buf[i+(j+1)*structure.SizeX]},
					},
				},
			)
		}
	}

	for i := 0; i < maxX-1; i++ {
		// Front
		solid.AppendTriangle(
			stl.Triangle{
				Vertices: [3]stl.Vec3{
					{float32(i), 0, 0},
					{float32(i + 1), 0, 0},
					{float32(i), 0, buf[i]},
				},
			},
		)
		solid.AppendTriangle(
			stl.Triangle{
				Vertices: [3]stl.Vec3{
					{float32(i + 1), 0, 0},
					{float32(i + 1), 0, buf[i+1]},
					{float32(i), 0, buf[i]},
				},
			},
		)

		// Back
		solid.AppendTriangle(
			stl.Triangle{
				Vertices: [3]stl.Vec3{
					{float32(i), float32(maxY), 0},
					{float32(i + 1), float32(maxY), 0},
					{float32(i), float32(maxY), buf[i+maxY*structure.SizeX]},
				},
			},
		)
		solid.AppendTriangle(
			stl.Triangle{
				Vertices: [3]stl.Vec3{
					{float32(i + 1), float32(maxY), 0},
					{float32(i + 1), float32(maxY), buf[i+1+maxY*structure.SizeX]},
					{float32(i), float32(maxY), buf[i+maxY*structure.SizeX]},
				},
			},
		)
	}

	for j := 0; j < maxY-1; j++ {
		// Left
		solid.AppendTriangle(
			stl.Triangle{
				Vertices: [3]stl.Vec3{
					{0, float32(j), 0},
					{0, float32(j + 1), 0},
					{0, float32(j), buf[j*structure.SizeY]},
				},
			},
		)
		solid.AppendTriangle(
			stl.Triangle{
				Vertices: [3]stl.Vec3{
					{0, float32(j + 1), 0},
					{0, float32(j + 1), buf[(j+1)*structure.SizeY]},
					{0, float32(j), buf[j*structure.SizeY]},
				},
			},
		)

		// Right
		solid.AppendTriangle(
			stl.Triangle{
				Vertices: [3]stl.Vec3{
					{float32(maxX), float32(j), 0},
					{float32(maxX), float32(j + 1), 0},
					{float32(maxX), float32(j), buf[maxX+j*structure.SizeY]},
				},
			},
		)
		solid.AppendTriangle(
			stl.Triangle{
				Vertices: [3]stl.Vec3{
					{float32(maxX), float32(j + 1), 0},
					{float32(maxX), float32(j + 1), buf[maxX+(j+1)*structure.SizeY]},
					{float32(maxX), float32(j), buf[maxX+j*structure.SizeY]},
				},
			},
		)
	}

	// Base
	solid.AppendTriangle(
		stl.Triangle{
			Vertices: [3]stl.Vec3{
				{0.0, 0.0, 0.0},
				{float32(maxX), 0.0, 0.0},
				{0.0, float32(maxY), 0.0},
			},
		},
	)
	solid.AppendTriangle(
		stl.Triangle{
			Vertices: [3]stl.Vec3{
				{float32(maxX), 0.0, 0.0},
				{float32(maxX), float32(maxY), 0.0},
				{0.0, float32(maxY), 0.0},
			},
		},
	)

	solid.RecalculateNormals()
	solid.Validate()
	solid.WriteFile("relief.stl")
	return
	/*
		adfMinMax[0] = GDALGetRasterMinimum( hBand, &bGotMin );
		adfMinMax[1] = GDALGetRasterMaximum( hBand, &bGotMax );
		if( ! (bGotMin && bGotMax) )
			GDALComputeRasterMinMax( hBand, TRUE, adfMinMax );
		printf( "Min=%.3fd, Max=%.3f\n", adfMinMax[0], adfMinMax[1] );
	*/
	//not implemented

	/*
		if( GDALGetOverviewCount(hBand) > 0 )
			printf( "Band has %d overviews.\n", GDALGetOverviewCount(hBand));
	*/
	if overviews := band.Overviews(); len(overviews) > 0 {
		fmt.Printf("Band has %d overviews.\n", len(overviews))
	}

	/*
		if( GDALGetRasterColorTable( hBand ) != NULL )
			printf( "Band has a color table with %d entries.\n",
					GDALGetColorEntryCount(
						GDALGetRasterColorTable( hBand ) ) );
	*/
	if ct := band.ColorTable(); len(ct.Entries) > 0 {
		fmt.Printf("Band has a color table with %d entries.\n", len(ct.Entries))
	}

	/*
		float *pafScanline;
		int   nXSize = GDALGetRasterBandXSize( hBand );
		pafScanline = (float *) CPLMalloc(sizeof(float)*nXSize);
		GDALRasterIO( hBand, GF_Read, 0, 0, nXSize, 1,
			pafScanline, nXSize, 1, GDT_Float32,
			0, 0 );
	*/

	pafScanline := make([]float32, structure.SizeX)
	err = band.Read(0, 0, pafScanline, bandStructure.SizeX, 1)
	if err != nil {
		panic(err)
	}

	err = hDataset.Close()
	// we don't really need to check for errors here as we have a read-only dataset.
	if err != nil {
		panic(err)
	}

	/*
		const char *pszFormat = "GTiff";
		GDALDriverH hDriver = GDALGetDriverByName( pszFormat );
		char **papszMetadata;
		if( hDriver == NULL )
		    exit( 1 );
		papszMetadata = GDALGetMetadata( hDriver, NULL );
		if( CSLFetchBoolean( papszMetadata, GDAL_DCAP_CREATE, FALSE ) )
		    printf( "Driver %s supports Create() method.\n", pszFormat );
		if( CSLFetchBoolean( papszMetadata, GDAL_DCAP_CREATECOPY, FALSE ) )
		    printf( "Driver %s supports CreateCopy() method.\n", pszFormat );
	*/

	hDriver, ok := godal.RasterDriver("Gtiff")
	if !ok {
		panic("Gtiff not found")
	}
	md := hDriver.Metadatas()
	if md["DCAP_CREATE"] == "YES" {
		fmt.Printf("Driver GTiff supports Create() method.\n")
	}
	if md["DCAP_CREATECOPY"] == "YES" {
		fmt.Printf("Driver Gtiff supports CreateCopy() method.\n")
	}

	/*	GDALDataset *poSrcDS = (GDALDataset *) GDALOpen( pszSrcFilename, GA_ReadOnly );
		GDALDataset *poDstDS;
		char **papszOptions = NULL;
		papszOptions = CSLSetNameValue( papszOptions, "TILED", "YES" );
		papszOptions = CSLSetNameValue( papszOptions, "COMPRESS", "PACKBITS" );
		poDstDS = poDriver->CreateCopy( pszDstFilename, poSrcDS, FALSE,
										papszOptions, GDALTermProgress, NULL );
		if( poDstDS != NULL )
			GDALClose( (GDALDatasetH) poDstDS );
		CSLDestroy( papszOptions );

		GDALClose( (GDALDatasetH) poSrcDS );
	*/

	//Left out: dealing with error handling
	poSrcDS, _ := godal.Open("testdata/test.tif")
	pszDstFilename := "/vsimem/tempfile.tif"
	defer godal.VSIUnlink(pszDstFilename)
	//godal doesn't expose createCopy directly, but the same result can be obtained with Translate
	poDstDS, _ := poSrcDS.Translate(pszDstFilename, nil, godal.CreationOption("TILED=YES", "COMPRESS=PACKBITS"), godal.GTiff)
	poDstDS.Close()
	poSrcDS.Close()

	/*
		GDALDataset *poDstDS;
		char **papszOptions = NULL;
		poDstDS = poDriver->Create( pszDstFilename, 512, 512, 1, GDT_Byte,
									papszOptions );
		double adfGeoTransform[6] = { 444720, 30, 0, 3751320, 0, -30 };
		OGRSpatialReference oSRS;
		char *pszSRS_WKT = NULL;
		GDALRasterBand *poBand;
		GByte abyRaster[512*512];
		poDstDS->SetGeoTransform( adfGeoTransform );
		oSRS.SetUTM( 11, TRUE );
		oSRS.SetWellKnownGeogCS( "NAD27" );
		oSRS.exportToWkt( &pszSRS_WKT );
		poDstDS->SetProjection( pszSRS_WKT );
		CPLFree( pszSRS_WKT );
		poBand = poDstDS->GetRasterBand(1);
		poBand->RasterIO( GF_Write, 0, 0, 512, 512,
						abyRaster, 512, 512, GDT_Byte, 0, 0 );
		GDALClose( (GDALDatasetH) poDstDS );
	*/

	poDstDS, _ = godal.Create(godal.GTiff, pszDstFilename, 1, godal.Byte, 512, 512)
	defer poDstDS.Close() //Close can be defered / called more than once (second+ calls are no-ops)

	poDstDS.SetGeoTransform([6]float64{444720, 30, 0, 3751320, 0, -30})

	//SetUTM and SetWellKnownGeogCS not implemented. godal allows populating
	// a SpatialRef from a WKT or PROJ4 string, or an epsg code
	sr, _ := godal.NewSpatialRefFromEPSG(4326)
	defer sr.Close()
	poDstDS.SetSpatialRef(sr)

	abyRaster := make([]byte, 512*512)
	// ... now populate with data
	poDstDS.Bands()[0].Write(0, 0, abyRaster, 512, 512)
	poDstDS.Close()

}
