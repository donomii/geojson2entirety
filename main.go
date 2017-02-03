package main

import (
	"log"
	//"strings"
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jharlap/geojson"
)
import "flag"

func writeBytes(n float64, f *os.File) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, n)
	if err != nil {
		log.Println("binary.Write failed:", err)
	}
	_, err = f.Write(buf.Bytes())
	//fmt.Printf("%s", buf.Bytes())
}

func writeBytesInt(n int64, f *os.File) {
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

func unpackJSON(accum []byte) geojson.Container {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Caught error in unpackJSON ", r)
			//return geojson.Container{}
		}
	}()
	result := geojson.Container{}
	err := json.Unmarshal(accum, &result)
	check(err)
	return result
}

func main() {
	var mapName = flag.String("outFile", "default_map", "Name for map file")
	var limit = flag.Int("limit", -1, "Limit the number of records imported")
	//var skip = flag.Int("skip", -1, "Skip every nth record")
	/*var mapName string
	if len(os.Args) > 1 {
		mapName = os.Args[1]
		log.Println(mapName)
	}*/

	flag.Parse()
	var err error
	scanner := bufio.NewReader(os.Stdin)
	tagpointsFile, err := os.Create(*mapName + ".tag_points")
	check(err)
	defer tagpointsFile.Close()
	pointsFile, err := os.Create(*mapName + ".map_points")
	check(err)
	defer pointsFile.Close()
	pointdataFile, err := os.Create(*mapName + ".map_data")
	check(err)
	defer pointdataFile.Close()
	tagcatFile, err := os.Create(*mapName + ".tag_category")
	check(err)
	defer tagcatFile.Close()
	preoffsetFile, err := os.Create(*mapName + ".pre_offset")
	check(err)
	defer preoffsetFile.Close()
	offsetFile, err := os.Create(*mapName + ".tag_offset")
	check(err)
	defer offsetFile.Close()
	stringsFile, err := os.Create(*mapName + ".tag_text")
	check(err)
	defer stringsFile.Close()
	indexFile, err := os.Create(*mapName + ".tag_index")
	check(err)
	defer indexFile.Close()
	line := []byte{}
	more := false
	accum := []byte{}
	offset := 0
	count := 0
	for {

		line, more, err = scanner.ReadLine()
		if err != nil {
			log.Printf("Imported %d records\n", count)
			log.Println("Done.  Finished map: ", *mapName)
			os.Exit(0)
		}
		if more {
			accum = append(accum, line...)
		} else {
			if *limit > -1 && count > *limit {
				log.Printf("Finishing import early after %d records for %v", count, *mapName)
				os.Exit(0)
			}
			count = count + 1
			accum = append(accum, line...)
			//fmt.Printf("Line: %v\n", string(accum[:len(accum)]))
			result := unpackJSON(accum)
			check(err)
			/*if *skip > -1 {
			if count >= *skip {
				count = 0
			} else {*/
			if result.Properties["name"] != nil && len(result.Properties["name"].(string)) > 1 {

				//fmt.Println("Parsed: ", string2Bytes(result.Properties["name"].(string)))
				//fmt.Printf("%s ", string2Bytes(result.Properties["name"].(string)))
				str := result.Properties["name"].(string)
				//log.Println("Adding tag: ", str)
				//str = strings.Replace(str, "\"", "\\\"", -1)
				outBytes, blength := string2Bytes(str)
				_, err = stringsFile.Write(outBytes)
				check(err)
				writeBytesInt(int64(offset), offsetFile)
				_, err = preoffsetFile.Write([]byte(fmt.Sprintf("%v\n", offset)))
				offset += blength

				writeBytes(result.Geometry.Point[1]*60, tagpointsFile)
				writeBytes(result.Geometry.Point[0]*-60, tagpointsFile)
				writeBytesInt(0, indexFile)
				writeBytesInt(0, tagcatFile)
			} else {
				//fmt.Println("Adding point without tag")
				writeBytes(result.Geometry.Point[1]*60, pointsFile)
				writeBytes(result.Geometry.Point[0]*-60, pointsFile)
				writeBytes(0, pointdataFile)
				writeBytes(0, pointdataFile)
				writeBytes(0, pointdataFile)
			}
			accum = []byte{}
		}
	}

	log.Println("Job's a good'un, boss!")
}
