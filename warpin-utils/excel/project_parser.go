package excel

import (
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/xuri/excelize/v2"
)

// --- Constants ---

// WebsiteRowHeight defines a consistent row height for website list items.
const WebsiteRowHeight = 25.0

// --- Template Structure Definition ---

type Field struct {
	DisplayName string
	JSONKey     string
	IsKeyValue  bool
}

var TemplateDefinition = []Field{
	{DisplayName: "Project Name", JSONKey: "name"},
	{DisplayName: "Training Purpose", JSONKey: "training_purpose"},
	{DisplayName: "Final Effect", JSONKey: "final_effect"},
	{DisplayName: "Expected Data Websites (Key-Value List)", JSONKey: "expected_data_websites", IsKeyValue: true},
}

// --- Data Structures ---

type ParsedProjectData struct {
	Name                 string
	TrainingPurpose      string
	FinalEffect          string
	ExpectedDataWebsites string
}

// WebsiteData defines the structure for a single website entry.
type WebsiteData struct {
	Site       string `json:"site"`
	ExpectData string `json:"expectData"`
}

// --- Core Functions ---

// BuildTemplateLayout constructs the styled layout in a new Excel file.
// This function now creates a template without any pre-allocated website data rows.
func BuildTemplateLayout(f *excelize.File) error {
	sheetName := f.GetSheetName(0)

	headerStyle, _ := createHeaderStyle(f)
	labelStyle, _ := createLabelStyle(f)
	inputStyle, _ := createInputStyle(f)
	noteStyle, _ := createNoteStyle(f)

	f.SetColWidth(sheetName, "A", "A", 5)
	f.SetColWidth(sheetName, "B", "B", 25)
	f.SetColWidth(sheetName, "C", "E", 30)

	currentRow := 6 // Start from row 6

	// Header Block
	f.MergeCell(sheetName, "B6", "E8")
	f.SetCellValue(sheetName, "B6", "Template Instructions:\n- Please fill in the project details in the designated spaces below.\n- For 'Expected Data Websites', data rows will be dynamically inserted.")
	f.SetCellStyle(sheetName, "B6", "E8", headerStyle)
	f.SetRowHeight(sheetName, 6, 60)
	currentRow = 10 // Start main fields from row 10

	// Main Project Fields
	mainFields := []string{"Project Name", "Training Purpose", "Final Effect"}
	for _, field := range mainFields {
		labelCell := fmt.Sprintf("B%d", currentRow)
		inputCellStart := fmt.Sprintf("C%d", currentRow)
		inputCellEnd := fmt.Sprintf("E%d", currentRow)
		f.SetCellValue(sheetName, labelCell, field)
		f.SetCellStyle(sheetName, labelCell, labelCell, labelStyle)
		f.MergeCell(sheetName, inputCellStart, inputCellEnd)
		f.SetCellValue(sheetName, inputCellStart, "")
		f.SetCellStyle(sheetName, inputCellStart, inputCellEnd, inputStyle)
		f.SetRowHeight(sheetName, currentRow, 25)
		currentRow++
	}

	currentRow += 2 // Gap

	// Expected Data Websites Header
	websiteHeaderCellStart := fmt.Sprintf("B%d", currentRow)
	websiteHeaderCellEnd := fmt.Sprintf("E%d", currentRow)
	f.MergeCell(sheetName, websiteHeaderCellStart, websiteHeaderCellEnd)
	f.SetCellValue(sheetName, websiteHeaderCellStart, "Expected Data Websites (Key-Value List)")
	f.SetCellStyle(sheetName, websiteHeaderCellStart, websiteHeaderCellEnd, headerStyle)
	currentRow++

	// Websites Sub-headers
	keyHeaderCellStart := fmt.Sprintf("B%d", currentRow)
	keyHeaderCellEnd := fmt.Sprintf("C%d", currentRow)
	valueHeaderCellStart := fmt.Sprintf("D%d", currentRow)
	valueHeaderCellEnd := fmt.Sprintf("E%d", currentRow)
	f.MergeCell(sheetName, keyHeaderCellStart, keyHeaderCellEnd)
	f.SetCellValue(sheetName, keyHeaderCellStart, "Key (e.g., Site Name)")
	f.SetCellStyle(sheetName, keyHeaderCellStart, keyHeaderCellEnd, labelStyle)
	f.MergeCell(sheetName, valueHeaderCellStart, valueHeaderCellEnd)
	f.SetCellValue(sheetName, valueHeaderCellStart, "Value (e.g., URL)")
	f.SetCellStyle(sheetName, valueHeaderCellStart, valueHeaderCellEnd, labelStyle)

	// Apply consistent row height to the sub-header row
	if err := f.SetRowHeight(sheetName, currentRow, WebsiteRowHeight); err != nil {
		log.Printf("Warning: Failed to set row height for website sub-header row %d: %v", currentRow, err)
	}
	currentRow++

	// Final instruction note (now directly follows sub-headers in the base template)
	noteCellStart := fmt.Sprintf("B%d", currentRow)
	noteCellEnd := fmt.Sprintf("E%d", currentRow)
	f.MergeCell(sheetName, noteCellStart, noteCellEnd)
	f.SetCellValue(sheetName, noteCellStart, "To add more websites, insert a new row above this note and apply the same formatting.")
	f.SetCellStyle(sheetName, noteCellStart, noteCellEnd, noteStyle)

	// Hide columns beyond E
	for col := 'F'; col <= 'X'; col++ {
		colName := string(col)
		if err := f.SetColVisible(sheetName, colName, false); err != nil {
			log.Printf("Warning: Failed to hide column %s: %v", colName, err)
		}
	}

	return nil
}

// WriteDataToTemplate dynamically writes data, inserting rows if necessary.
func WriteDataToTemplate(excelFile *excelize.File, formData map[string]string) error {
	if len(formData) == 0 {
		return nil
	}

	sheetName := excelFile.GetSheetName(0)
	rows, err := excelFile.GetRows(sheetName)
	if err != nil {
		return fmt.Errorf("failed to get rows: %w", err)
	}

	fieldMap := make(map[string]Field)
	for _, field := range TemplateDefinition {
		fieldMap[field.DisplayName] = field
	}

	inputStyle, _ := createInputStyle(excelFile)
	websiteInputStyle, _ := createWebsiteInputStyle(excelFile)

	websiteDataInsertRow := -1 // The row number where website data should start being inserted
	noteRow := -1              // The row number of the final instruction note

	// First pass: find the insertion point for website data and the note row.
	for i, row := range rows {
		if len(row) > 1 {
			displayName := row[1]
			if displayName == "Key (e.g., Site Name)" { // Found the sub-header for websites
				websiteDataInsertRow = i + 2 // Data starts 1 row below sub-headers
			}
			if displayName == "To add more websites, insert a new row above this note and apply the same formatting." {
				noteRow = i + 1 // Note is at this row index + 1
			}
		}
	}

	if websiteDataInsertRow == -1 || noteRow == -1 {
		return fmt.Errorf("could not find website data insertion point or note row in template")
	}

	// Write simple fields
	for i, row := range rows {
		if len(row) > 1 {
			displayName := row[1]
			if field, ok := fieldMap[displayName]; ok && !field.IsKeyValue {
				if value, dataOk := formData[field.JSONKey]; dataOk && value != "" {
					cell := fmt.Sprintf("C%d", i+1)
					if err := excelFile.SetCellValue(sheetName, cell, value); err != nil {
						return err
					}
					excelFile.SetCellStyle(sheetName, cell, fmt.Sprintf("E%d", i+1), inputStyle)
				}
			}
		}
	}

	// Handle Expected Data Websites (Key-Value List)
	if websitesJSON, ok := formData["expected_data_websites"]; ok && websitesJSON != "" {
		var websites []WebsiteData
		if err := json.Unmarshal([]byte(websitesJSON), &websites); err == nil {
			numWebsites := len(websites)

			// If there are websites to write, insert rows for them.
			if numWebsites > 0 {
				// The insertion point is the row where the note currently is.
				// We insert 'numWebsites' rows at 'noteRow'.
				if err := excelFile.InsertRows(sheetName, noteRow, numWebsites); err != nil {
					return fmt.Errorf("failed to insert rows for websites: %w", err)
				}

				// After inserting rows, the noteRow has shifted. Update it.
				noteRow += numWebsites

				// Write each website entry into the Excel rows
				for i, siteData := range websites {
					rowNum := websiteDataInsertRow + i // This is the actual row number for the data
					keyCellStart := fmt.Sprintf("B%d", rowNum)
					keyCellEnd := fmt.Sprintf("C%d", rowNum)
					valueCellStart := fmt.Sprintf("D%d", rowNum)
					valueCellEnd := fmt.Sprintf("E%d", rowNum)

					// Merge cells
					excelFile.MergeCell(sheetName, keyCellStart, keyCellEnd)
					excelFile.MergeCell(sheetName, valueCellStart, valueCellEnd)

					// Set values.
					excelFile.SetCellValue(sheetName, keyCellStart, siteData.Site)
					excelFile.SetCellValue(sheetName, valueCellStart, siteData.ExpectData)

					// Apply style to each merged cell range
					if err := excelFile.SetCellStyle(sheetName, keyCellStart, keyCellEnd, websiteInputStyle); err != nil {
						return fmt.Errorf("failed to set style for key cell range %s:%s: %w", keyCellStart, keyCellEnd, err)
					}
					if err := excelFile.SetCellStyle(sheetName, valueCellStart, valueCellEnd, websiteInputStyle); err != nil {
						return fmt.Errorf("failed to set style for value cell range %s:%s: %w", valueCellStart, valueCellEnd, err)
					}
					// Apply consistent row height
					if err := excelFile.SetRowHeight(sheetName, rowNum, WebsiteRowHeight); err != nil {
						return fmt.Errorf("failed to set row height for row %d: %w", rowNum, err)
					}
				}
			}
		}
	}

	return nil
}

// ParseProjectTemplate ... (remains the same)
func ParseProjectTemplate(formData map[string]string, reader io.Reader) (*ParsedProjectData, error) {
	data := &ParsedProjectData{
		Name:                 formData["name"],
		TrainingPurpose:      formData["training_purpose"],
		FinalEffect:          formData["final_effect"],
		ExpectedDataWebsites: formData["expected_data_websites"],
	}

	if reader == nil {
		return data, nil
	}

	excelFile, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read excel file from reader: %w", err)
	}

	rows, err := excelFile.GetRows(excelFile.GetSheetName(0))
	if err != nil {
		return nil, fmt.Errorf("failed to get excel rows: %w", err)
	}

	websites := []WebsiteData{}
	parsingWebsites := false

	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		key := row[1]

		if !parsingWebsites {
			var value string
			if len(row) > 2 {
				value = row[2]
			}

			switch key {
			case "Project Name":
				if value != "" {
					data.Name = value
				}
			case "Training Purpose":
				if value != "" {
					data.TrainingPurpose = value
				}
			case "Final Effect":
				if value != "" {
					data.FinalEffect = value
				}
			case "Expected Data Websites (Key-Value List)":
				parsingWebsites = true
			}
		} else {
			if key == "" || key == "To add more websites, insert a new row above this note and apply the same formatting." {
				parsingWebsites = false
				continue
			}
			if key == "Key (e.g., Site Name)" {
				continue
			}

			var site, expectData string
			if len(row) > 1 {
				site = row[1]
			} // Key/Site is in merged B:C
			if len(row) > 3 {
				expectData = row[3]
			} // Value/ExpectData is in merged D:E

			if site != "" || expectData != "" {
				websites = append(websites, WebsiteData{Site: site, ExpectData: expectData})
			}
		}
	}

	if len(websites) > 0 {
		websitesJSON, err := json.Marshal(websites)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal websites data to JSON: %w", err)
		}
		data.ExpectedDataWebsites = string(websitesJSON)
	}

	return data, nil
}
