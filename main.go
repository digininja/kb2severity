package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

func getPage(page string) (string, error) {
	resp, err := http.Get(page)
	if err != nil {
		fmt.Printf("Could not connect: %s\n", err)
	} else {
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading body: %s\n", err)
			return "", err
		} else {
			// fmt.Printf("Body: %s\n", string(body))
			return string(body), nil
		}
	}
	return "", errors.New("Could not get response")
}

type KB_details struct {
	kb          string
	original_kb string
	blog_url    string
	catalog_url string
	details_url string
	severity    string
}

func (kb_details KB_details) Print() {
	fmt.Printf("Original KB: %s\n", kb_details.original_kb)
	fmt.Printf("Current KB: %s\n", kb_details.kb)
	fmt.Printf("Blog URL: %s\n", kb_details.blog_url)
	fmt.Printf("Catalog URL: %s\n", kb_details.catalog_url)
	fmt.Printf("Details URL: %s\n", kb_details.details_url)
	fmt.Printf("Severity: %s\n", kb_details.severity)
}

func CSVHeader() []string {
	strings := make([]string, 6)

	strings[0] = "Original KB"
	strings[1] = "Current KB"
	strings[2] = "Blog URL"
	strings[3] = "Catalog URL"
	strings[4] = "Details URL"
	strings[5] = "Severity"

	return strings
}

func (kb_details KB_details) AsCSV() []string {
	strings := make([]string, 6)

	strings[0] = kb_details.original_kb
	strings[1] = kb_details.kb
	strings[2] = kb_details.blog_url
	strings[3] = kb_details.catalog_url
	strings[4] = kb_details.details_url
	strings[5] = kb_details.severity

	return strings
}

func main() {
	output_csv_name := "out.csv"
	input_name := "kbs.txt"

	var kbs = []string{}

	file, err := os.Open(input_name)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		kbs = append(kbs, scanner.Text())
		// fmt.Printf("Line: %s\n", scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	var all_results = []KB_details{}

	for _, kb := range kbs {
		kb_details := KB_details{}
		kb_details.original_kb = kb
		just_number := strings.Trim(kb, "KB")

		blog_url := fmt.Sprintf("https://support.microsoft.com/en-us/help/%s/windows-10-update-%s", just_number, kb)
		blog_page, err := getPage(blog_url)
		// fmt.Printf("Blog url: %s\n", blog_url)
		kb_details.blog_url = blog_url

		if err != nil {
			log.Print(fmt.Sprintf("Could not connect to blog page for %s: %s\n", kb, err))
			log.Print(fmt.Sprintf("URL: %s\n", blog_url))
		} else {
			var find_kb_re = regexp.MustCompile(`<a href=.*q=(KB[0-9]*).*Microsoft Update Catalog`)

			// fmt.Printf("blog page %s\n", blog_page)
			current_kb_hit := find_kb_re.FindStringSubmatch(blog_page)
			if len(current_kb_hit) == 2 {
				current_kb := current_kb_hit[1]
				kb_details.kb = current_kb

				catalog_page_url := fmt.Sprintf("http://www.catalog.update.microsoft.com/Search.aspx?q=%s", current_kb)
				// fmt.Printf("Catalog URL: %s\n", catalog_page_url)
				kb_details.catalog_url = catalog_page_url

				catalog_page, err := getPage(catalog_page_url)

				if err != nil {
					log.Print(fmt.Sprintf("Could not connect to catalog page for %s: %s\n", kb, err))
					log.Print(fmt.Sprintf("URL: %s\n", catalog_page_url))
				} else {
					var find_link_re = regexp.MustCompile(`<a id='([^']*)_link'.*goToDetails.*>`)

					link := find_link_re.FindStringSubmatch(catalog_page)
					if len(link) == 2 {
						link_id := link[1]
						// fmt.Printf("link: %s\n", link_id)
						link_url := fmt.Sprintf("http://www.catalog.update.microsoft.com/ScopedViewInline.aspx?updateid=%s", link_id)
						kb_details.details_url = link_url
						// fmt.Printf("link URL: %s\n", link_url)

						details_page, err := getPage(link_url)
						if err != nil {
							log.Print(fmt.Sprintf("Could not connect to details page for %s: %s\n", kb, err))
							log.Print(fmt.Sprintf("URL: %s\n", link_url))
						} else {

							var find_severity_re = regexp.MustCompile(`<.*ScopedViewHandler_msrcSeverity">([^<]*)</span`)

							details := find_severity_re.FindStringSubmatch(details_page)
							if len(details) == 2 {
								severity := details[1]
								kb_details.severity = severity

								// fmt.Printf("Severity: %s\n", severity)

								kb_details.Print()
							} else {
								log.Print(fmt.Sprintf("Could not find the severity in the page for %s\n", kb))
								log.Print(fmt.Sprintf("URL: %s\n", link_url))
							}

							// <span id="ScopedViewHandler_labelMSRCSeverity_Separator" class="labelTitle">MSRC severity:</span>
						}
					} else {
						log.Print(fmt.Sprintf("Could not find link in page for %s\n", kb))
						log.Print(fmt.Sprintf("URL: %s\n", catalog_page_url))
					}
				}
			} else {
				log.Print(fmt.Sprintf("Could not find link in blog page for %s\n", kb))
				log.Print(fmt.Sprintf("URL: %s\n", blog_url))
			}
		}
		all_results = append(all_results, kb_details)
	}

	out_file, err := os.Create(output_csv_name)
	if err != nil {
		log.Fatal(err)
	}
	defer out_file.Close()

	w := csv.NewWriter(out_file)

	if err := w.Write(CSVHeader()); err != nil {
		fmt.Printf("error writing record to csv: %s", err)
	}
	for _, record := range all_results {
		if err := w.Write(record.AsCSV()); err != nil {
			fmt.Printf("error writing record to csv: %s", err)
		}
	}

	// Write any buffered data to the underlying writer (standard output).
	w.Flush()

	if err := w.Error(); err != nil {
		fmt.Printf("Error flushing buffer: %s", err)
	}
}
