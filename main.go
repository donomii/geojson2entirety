package main

import (
	"log"

	//"github.com/tchap/go-patricia/patricia"

	//"strings"
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"

	"flag"

	"github.com/jharlap/geojson"
)

func writeBytes(n float64, f *bufio.Writer) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, n)
	if err != nil {
		log.Println("binary.Write failed:", err)
	}
	_, err = f.Write(buf.Bytes())
	//fmt.Printf("%s", buf.Bytes())
}

func writeBytesInt(n int64, f *bufio.Writer) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, n)
	if err != nil {
		log.Println("binary.Write failed:", err)
	}
	_, err = f.Write(buf.Bytes())
	//fmt.Printf("%s", buf.Bytes())
}

func string2Bytes(s string) ([]byte, int) {
	b := []byte(s)
	b = append(b, []byte{0}...)
	l := len(b)
	return b, l
}

func check(err error) {
	if err != nil {
		log.Println(err)
	}
}

func checkJSONerr(err error, js []byte) {
	if err != nil {
		log.Println(err)
		log.Println(string(js))
	}
}

func unpackJSON(accum []byte) (geojson.Container, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Caught error in unpackJSON ", r)
			//return geojson.Container{}, nil
		}
	}()
	result := geojson.Container{}
	err := json.Unmarshal(accum, &result)
	checkJSONerr(err, accum)
	if err != nil {
		return geojson.Container{}, err
	}
	return result, nil
}

func writeTag(str string, long, lat float64, tagpointsFile, offsetFile, indexFile, tagcatFile, stringsFile, preoffsetFile *bufio.Writer, indexCount, offset int64) int64 {
	//treeIndexAdd2(str, long, lat)
	//fmt.Println("Parsed: ", string2Bytes(result.Properties["name"].(string)))
	//fmt.Printf("%s ", string2Bytes(result.Properties["name"].(string)))

	//str = strings.Replace(str, "\"", "\\\"", -1)
	if verbose {
		log.Println("Adding tag ", indexCount, ": ", str, " at ", lat, ",", long, " at offset ", offset)
	}
	outBytes, blength := string2Bytes(str)
	wrote, err := stringsFile.Write(outBytes)
	if wrote != blength {
		panic("Written length is different from string length!")
	}
	check(err)
	writeBytesInt(int64(offset), offsetFile)
	_, err = preoffsetFile.Write([]byte(fmt.Sprintf("%v\n", offset)))

	writeBytes(lat, tagpointsFile)
	writeBytes(long, tagpointsFile)
	writeBytesInt(indexCount-1, indexFile)
	writeBytesInt(0, tagcatFile)
	return int64(blength)
}

var verbose bool

func openFile(mapname string) (*os.File, *bufio.Writer) {
	f, err := os.Create(mapname)
	check(err)
	//defer tagpointsFile.Close()
	w := bufio.NewWriterSize(f, 10*1024*1024)
	return f, w
}

func main() {
	var mapName = flag.String("outFile", "default_map", "Name for map file")
	var limit = flag.Int64("limit", -1, "Limit the number of records imported")
	var pointsOnly = flag.Bool("points", false, "Only save data point")
	var tagsOnly = flag.Bool("tags", false, "Only save tags(named points)")
	verbose = *flag.Bool("verbose", false, "Print progress")
	//var skip = flag.Int("skip", -1, "Skip every nth record")

	flag.Parse()

	if *tagsOnly {
		log.Println("Not writing points")
	}
	if *pointsOnly {
		log.Println("Not writing tags")
	}

	log.Println("Reading from stdin")
	var err error
	scanner := bufio.NewReader(os.Stdin)

	tp_handle, tagpointsFile := openFile(*mapName + ".tag_points")
	defer tagpointsFile.Flush()
	defer tp_handle.Close()

	pf_handle, pointsFile := openFile(*mapName + ".map_points")
	defer pointsFile.Flush()
	defer pf_handle.Close()

	pd_handle, pointdataFile := openFile(*mapName + ".map_data")
	defer pointdataFile.Flush()
	defer pd_handle.Close()

	tg_handle, tagcatFile := openFile(*mapName + ".tag_category")
	defer tagcatFile.Flush()
	defer tg_handle.Close()

	po_handle, preoffsetFile := openFile(*mapName + ".pre_offset")
	defer preoffsetFile.Flush()
	defer po_handle.Close()

	of_handle, offsetFile := openFile(*mapName + ".tag_offset")
	defer offsetFile.Flush()
	defer of_handle.Close()

	str_handle, stringsFile := openFile(*mapName + ".tag_text")
	defer stringsFile.Flush()
	defer str_handle.Close()

	in_handle, indexFile := openFile(*mapName + ".tag_index")
	defer indexFile.Flush()
	defer in_handle.Close()

	line := []byte{}
	more := false
	accum := []byte{}
	offset := int64(0)
	count := int64(0)
	indexCount := int64(0)
	offset += writeTag("FAIL", -60000, -6000, tagpointsFile, offsetFile, indexFile, tagcatFile, stringsFile, preoffsetFile, indexCount, offset)
	for {

		line, more, err = scanner.ReadLine()
		if err != nil {
			buildFinal()
			log.Printf("Imported %d records\n", count)
			log.Println("Done.  Finished map: ", *mapName)
			break
		}
		accum = append(accum, line...)
		if more {
			//accum = append(accum, line...)
		} else {
			if *limit > -1 && count > *limit {
				log.Printf("Finishing import early after %d records for %v", count, *mapName)
				buildFinal()
				os.Exit(0)
			}

			//accum = append(accum, line...)
			//fmt.Printf("Line: %v\n", string(accum[:len(accum)]))
			//log.Println("Unpacking", accum)
			result, err := unpackJSON(accum)
			check(err)
			if err == nil {

				/*if *skip > -1 {
				if count >= *skip {
					count = 0
				} else {*/
				if result.Properties["name"] != nil && len(result.Properties["name"].(string)) > 1 {
					count = count + 1
					indexCount += 1
					str := result.Properties["name"].(string)
					if !*pointsOnly {
						offset += writeTag(str, result.Geometry.Point[0]*-60, result.Geometry.Point[1]*60, tagpointsFile, offsetFile, indexFile, tagcatFile, stringsFile, preoffsetFile, indexCount, offset)
					}
				} else {
					if !*tagsOnly {
						if verbose {
							fmt.Println("Adding point without tag at ", result.Geometry.Point)
						}
						//treeIndexAdd2("", result.Geometry.Point[1]*-60, result.Geometry.Point[0]*60)
						writeBytes(result.Geometry.Point[1]*60, pointsFile)
						writeBytes(result.Geometry.Point[0]*-60, pointsFile)
						writeBytes(0, pointdataFile)
						writeBytes(0, pointdataFile)
						writeBytes(0, pointdataFile)
					}
				}
				accum = []byte{}
			} else {
				accum = []byte{}
			}
		}
	}
	//buildFinal()
	//iterateMp(mp)
	jsonString, err := json.MarshalIndent(mp, "", "  ")
	fmt.Println(err)
	fmt.Println(string(jsonString))
	/*tree2.Visit(func(prefix patricia.Prefix, item patricia.Item) error {
		fmt.Printf("lat: %v, lon: %v, data: %v\n", string(prefix), string(prefix), string(item.(string)))
		return nil
	})*/
	log.Println("Job's a good'un, boss!")
}
