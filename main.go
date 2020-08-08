package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
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
	catalog_url string
	details_url string
	severity    string
}

func (kb_details KB_details) Print() {
	fmt.Printf("Original KB: %s\n", kb_details.original_kb)
	fmt.Printf("Current KB: %s\n", kb_details.kb)
	fmt.Printf("Catalog URL: %s\n", kb_details.catalog_url)
	fmt.Printf("Details URL: %s\n", kb_details.details_url)
	fmt.Printf("Severity: %s\n", kb_details.severity)
}

func main() {
	kb := "KB4343887"
	kb = "KB2465373"
	kb = "KB4343887"

	// Have to do this page first to get the catalog URL as this page can redirect to a different KB
	// "https://support.microsoft.com/en-us/help/4343887/windows-10-update-KB2465373"

	kb_details := KB_details{}
	kb_details.original_kb = kb

	catalog_page_url := fmt.Sprintf("http://www.catalog.update.microsoft.com/Search.aspx?q=%s", kb)
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
				} else {
					fmt.Printf("Could not find the severity in the page\n")
				}

				// <span id="ScopedViewHandler_labelMSRCSeverity_Separator" class="labelTitle">MSRC severity:</span>
			}
		} else {
			fmt.Printf("Could not find link in page\n")
		}
	}
}
