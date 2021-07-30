package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"

	"net/http"

	"path"

	"github.com/gorilla/mux"
	shell "github.com/ipfs/go-ipfs-api"
	//mh "github.com/multiformats/go-multihash"
)

var sh *shell.Shell

type filePath struct {
	FilePath string `json:"FilePath"`
}

type fileName struct {
	FileName string `json:"FileName"`
}

type Details struct {
	Filename, Hash string
}

type DetailsDb struct {
	Filename string `json:"FileName"`
	Hash     string `json:"Hash"`
}

func find(records [][]string, val string, col int) (presence bool, hash string) {
	for _, row := range records {
		if row[col] == val {
			return true, row[1]
		}
	}
	return false, ""
}

func ProcessingFiles(destinationFile string) string {

	content, err := ioutil.ReadFile(destinationFile)
	text := string(content)
	fmt.Println(text)
	sh = shell.NewShell("localhost:5001")
	hash, err := sh.Add(bytes.NewBufferString(text))
	returnString := destinationFile + " " + "hash:" + hash
	if err != nil {
		fmt.Println("Error when uploading ipfs:", err)
	}
	fmt.Println("hash of the file:", hash)

	//writing as csv

	var toWrite [][]string

	var row []string
	row = append(row, destinationFile)
	row = append(row, hash)

	toWrite = append(toWrite, row)

	fmt.Println("toWrite:", toWrite)
	f, err := os.OpenFile("output.csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}
	err = csv.NewWriter(f).WriteAll(toWrite)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}
	return returnString
}

func createIPFSFile(w http.ResponseWriter, r *http.Request) {
	var fPath filePath

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Kindly enter file path in correct format")
	}

	json.Unmarshal(reqBody, &fPath)
	fmt.Println("path:", fPath.FilePath)
	PathString := fPath.FilePath
	dir, file := path.Split(PathString)
	extension := filepath.Ext(PathString)
	noExtfile := file[0 : len(file)-len(extension)]
	fmt.Println("PathString:", PathString)
	fmt.Println("dir:", dir)
	fmt.Println("file:", file)
	fmt.Println("noExtfile:", noExtfile)

	if len(filepath.Ext(PathString)) == 0 {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode("Directories cannot be used")

	} else {

		csvVar, err := os.Open("output.csv")
		if err != nil {
			log.Fatal(err)
		}

		records, err := csv.NewReader(csvVar).ReadAll()
		if err != nil {
			panic(err)
		}

		presence, hashVal := find(records, file, 0)
		fmt.Println("presence,hashval", presence, hashVal)

		if presence == true {

			inputFile, err := ioutil.ReadFile(PathString)
			if err != nil {
				fmt.Println(err)
				return
			}

			updVerString := strconv.Itoa(rand.Intn(100))
			fmt.Println("updVerString:", updVerString)

			destinationFile := noExtfile + "_" + updVerString + extension
			fmt.Println("destinationFile:", destinationFile)
			err = ioutil.WriteFile(destinationFile, inputFile, 0644)
			if err != nil {
				fmt.Println("Error creating", destinationFile)
				fmt.Println(err)
				return
			}

			returnString := ProcessingFiles(destinationFile)

			w.WriteHeader(http.StatusCreated)

			json.NewEncoder(w).Encode(returnString)

		} else {

			returnString := ProcessingFiles(PathString)

			w.WriteHeader(http.StatusCreated)

			json.NewEncoder(w).Encode(returnString)
		}

	}
}

func readIPFSFile(w http.ResponseWriter, r *http.Request) {
	var file fileName

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Kindly enter file path in correct format")
	}

	json.Unmarshal(reqBody, &file)
	fmt.Println("file:", file.FileName)
	fileName := file.FileName
	fmt.Println("filename:", fileName)

	csvVar, err := os.Open("output.csv")
	if err != nil {
		log.Fatal(err)
	}

	records, err := csv.NewReader(csvVar).ReadAll()
	if err != nil {
		panic(err)
	}

	presence, hashVal := find(records, fileName, 0)
	fmt.Println("presence,hashval", presence, hashVal)

	if presence == false {
		w.WriteHeader(http.StatusCreated)
		returnVal := fileName + " does not exist"
		json.NewEncoder(w).Encode(returnVal)
	} else {

		sh = shell.NewShell("localhost:5001")
		hash := hashVal
		//multihash, err := mh.FromB58String(hash)
		//fmt.Println("multihash:", string(multihash))
		if err != nil {
			fmt.Println(err)
		}

		read, err := sh.Cat(string(hash))
		if err != nil {
			fmt.Println(err)
		}
		body, err := ioutil.ReadAll(read)

		fmt.Println("content :", string(body))
		w.WriteHeader(http.StatusCreated)
		returnVal := string(body)
		json.NewEncoder(w).Encode(returnVal)
	}
}

func listIPFSFile(w http.ResponseWriter, r *http.Request) {
	csvVar, err := os.Open("output.csv")
	if err != nil {
		log.Fatal(err)
	}
	//csvReader := csv.NewReader(csvVar)
	records, err := csv.NewReader(csvVar).ReadAll()
	if err != nil {
		panic(err)
	}
	//fmt.Println("records:", string(records))
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(records)
}

func main() {

	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/writeIPFS", createIPFSFile).Methods("POST")
	router.HandleFunc("/readIPFS", readIPFSFile).Methods("POST")
	router.HandleFunc("/listIPFS", listIPFSFile).Methods("POST")

	log.Fatal(http.ListenAndServe(":8082", router))
}
