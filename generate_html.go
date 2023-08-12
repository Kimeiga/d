package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type Entry struct {
	Traditional string   `json:"traditional"`
	Simplified  string   `json:"simplified"`
	Pinyin      string   `json:"pinyin"`
	Definitions []string `json:"definitions"`
}

func createHTML(entry Entry, folderPath string) {
	filename := filepath.Join(folderPath, entry.Simplified+".html")
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	fmt.Fprintf(writer, "<h1>%s</h1>\r\n<p>Simplified: %s</p>\r\n<p>Traditional: %s</p>\r\n<p>Pinyin: %s</p>", entry.Simplified, entry.Simplified, entry.Traditional, entry.Pinyin)
	for _, def := range entry.Definitions {
		fmt.Fprintf(writer, "<p>%s</p>\r\n", def)
	}
	writer.Flush()
}

func deleteFilesInFolder(folderPath string) {
	files, err := filepath.Glob(filepath.Join(folderPath, "*.html"))
	if err != nil {
		fmt.Println("Error fetching files:", err)
		return
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			fmt.Println("Error deleting file:", err)
		}
	}
}

func printProgress(progress, total int) {
	// Calculate percentage
	percent := float64(progress) / float64(total) * 100
	// Calculate number of blocks to represent progress
	blocks := int(percent / 2)

	// Print the progress bar
	fmt.Printf("\r[")
	for i := 0; i < blocks; i++ {
		fmt.Print("=")
	}
	for i := 0; i < 50-blocks; i++ {
		fmt.Print(" ")
	}
	fmt.Printf("] %3.0f%%", percent)

	if progress == total {
		fmt.Println() // Print a newline at 100%
	}
}

func main() {
	buildFolder := "docs"

	err := os.MkdirAll(buildFolder, os.ModePerm)
	if err != nil {
		fmt.Println("Error creating build folder:", err)
		return
	}

	// Delete existing HTML files
	deleteFilesInFolder(buildFolder)

	indexFile, err := os.Create(filepath.Join(buildFolder, "index.html"))
	if err != nil {
		fmt.Println("Error creating index file:", err)
		return
	}
	defer indexFile.Close()

	indexWriter := bufio.NewWriter(indexFile)
	fmt.Fprint(indexWriter, `<!DOCTYPE html>
<html>
<head>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		.grid-container {
			display: flex;
			flex-wrap: wrap;
		}
		.grid-item {
			width: 100px;
			height: 100px;
			display: flex;
			align-items: center;
			justify-content: center;
			border: 1px solid black;
			flex-grow: 1;
			text-align: center;
		}
		@media (max-width: 600px) {
			.grid-item {
				width: 50px;
				height: 50px;
			}
		}
	</style>
</head>
<body>
	<div class="grid-container">
`)

	file, err := os.Open("cedict.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	var entries []Entry
	err = json.Unmarshal(bytes, &entries)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	total := len(entries) // Get the total number of entries
	var progress int
	var printMutex sync.Mutex

	// Create a semaphore channel to limit concurrent goroutines
	sem := make(chan bool, 64) // 20 is the number of concurrent goroutines allowed

	var wg sync.WaitGroup

	for _, entry := range entries {
		wg.Add(1)
		sem <- true // Acquire a token
		go func(e Entry) {
			defer wg.Done()
			createHTML(e, buildFolder)

			progress++
			printMutex.Lock()
			printProgress(progress, total)
			printMutex.Unlock()

			<-sem // Release the token
		}(entry)

		// Add the link to the index file
		fmt.Fprintf(indexWriter, `<a class="grid-item" href="%s.html">%s</a>`+"\n", entry.Simplified, entry.Simplified)
	}

	fmt.Fprint(indexWriter, `
	</div>
</body>
</html>
`)
	indexWriter.Flush()

	wg.Wait()
}
