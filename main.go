package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
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
	var kbs = []string{}
	kbs = append(kbs, "KB4343887")
	kbs = append(kbs, "KB2465373")

	var all_results = []KB_details{}

	for _, kb := range kbs {

		kb_details := KB_details{}
		kb_details.original_kb = kb

		blog_url := fmt.Sprintf("https://support.microsoft.com/en-us/help/4343887/windows-10-update-%s", kb)
		blog_page, err := getPage(blog_url)
		// fmt.Printf("Blog url: %s\n", blog_url)
		kb_details.blog_url = blog_url

		if err != nil {
			fmt.Printf("Could not connect to blog page: %s\n", err)
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
					fmt.Printf("Could not connect to catalog page: %s\n", err)
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
							fmt.Printf("Could not connect to details page: %s\n", err)
						} else {

							var find_severity_re = regexp.MustCompile(`<.*ScopedViewHandler_msrcSeverity">([^<]*)</span`)

							details := find_severity_re.FindStringSubmatch(details_page)
							if len(details) == 2 {
								severity := details[1]
								kb_details.severity = severity

								// fmt.Printf("Severity: %s\n", severity)

								kb_details.Print()

								all_results = append(all_results, kb_details)

							} else {
								fmt.Printf("Could not find the severity in the page\n")
							}

							// <span id="ScopedViewHandler_labelMSRCSeverity_Separator" class="labelTitle">MSRC severity:</span>
						}
					} else {
						fmt.Printf("Could not find link in page\n")
					}
				}
			} else {
				fmt.Println("Could not find link in blog page")
			}
		}

	}

	w := csv.NewWriter(os.Stdout)

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
