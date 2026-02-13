package reports

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
)

//go:embed assets/fonts/DejaVuSans.ttf
var embeddedFont []byte

// Generator generates PDF/CSV reports
type Generator struct {
	metricsStorage  storage.MetricsStorage
	checkinsStorage CheckinsStorage
	profileStorage  ProfileStorage
}

// NewGenerator creates a new report generator
func NewGenerator(metricsStorage storage.MetricsStorage, checkinsStorage CheckinsStorage, profileStorage ProfileStorage) *Generator {
	return &Generator{
		metricsStorage:  metricsStorage,
		checkinsStorage: checkinsStorage,
		profileStorage:  profileStorage,
	}
}

// GenerateReport generates a report and returns the data
func (g *Generator) GenerateReport(ctx context.Context, req CreateReportRequest) ([]byte, error) {
	// Validate profile exists
	_, err := g.profileStorage.GetProfile(ctx, req.ProfileID)
	if err != nil {
		return nil, fmt.Errorf("profile not found")
	}

	// Fetch data
	dailyMetrics, err := g.metricsStorage.GetDailyMetrics(ctx, req.ProfileID, req.From, req.To)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch daily metrics: %w", err)
	}

	checkins, err := g.checkinsStorage.ListCheckins(ctx, req.ProfileID, req.From, req.To)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch checkins: %w", err)
	}

	// Generate based on format
	switch req.Format {
	case FormatPDF:
		return g.generatePDF(req, dailyMetrics, checkins)
	case FormatCSV:
		return g.generateCSV(req, dailyMetrics, checkins)
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}
}

// generateCSV generates a CSV report
func (g *Generator) generateCSV(req CreateReportRequest, dailyMetrics []storage.DailyMetricRow, checkins []Checkin) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Write header (English)
	header := []string{"date", "steps", "weight_kg_last", "resting_hr_bpm", "sleep_total_minutes", "morning_score", "evening_score"}
	if err := w.Write(header); err != nil {
		return nil, err
	}

	// Create a map for quick checkin lookup
	checkinsByDate := make(map[string]map[string]int) // date -> type -> score
	for _, c := range checkins {
		if checkinsByDate[c.Date] == nil {
			checkinsByDate[c.Date] = make(map[string]int)
		}
		checkinsByDate[c.Date][c.Type] = c.Score
	}

	// Write rows
	for _, dm := range dailyMetrics {
		// Parse JSON payload to extract fields
		var payload map[string]interface{}
		if err := json.Unmarshal(dm.Payload, &payload); err != nil {
			continue
		}

		row := []string{dm.Date}

		// Steps
		if activity, ok := payload["activity"].(map[string]interface{}); ok {
			if steps, ok := activity["steps"].(float64); ok {
				row = append(row, strconv.Itoa(int(steps)))
			} else {
				row = append(row, "")
			}
		} else {
			row = append(row, "")
		}

		// Weight
		if body, ok := payload["body"].(map[string]interface{}); ok {
			if weight, ok := body["weight_kg_last"].(float64); ok {
				row = append(row, fmt.Sprintf("%.2f", weight))
			} else {
				row = append(row, "")
			}
		} else {
			row = append(row, "")
		}

		// Resting HR
		if heart, ok := payload["heart"].(map[string]interface{}); ok {
			if hr, ok := heart["resting_hr_bpm"].(float64); ok {
				row = append(row, strconv.Itoa(int(hr)))
			} else {
				row = append(row, "")
			}
		} else {
			row = append(row, "")
		}

		// Sleep
		if sleep, ok := payload["sleep"].(map[string]interface{}); ok {
			if totalMin, ok := sleep["total_minutes"].(float64); ok {
				row = append(row, strconv.Itoa(int(totalMin)))
			} else {
				row = append(row, "")
			}
		} else {
			row = append(row, "")
		}

		// Morning score
		if morningScore, ok := checkinsByDate[dm.Date]["morning"]; ok {
			row = append(row, strconv.Itoa(morningScore))
		} else {
			row = append(row, "")
		}

		// Evening score
		if eveningScore, ok := checkinsByDate[dm.Date]["evening"]; ok {
			row = append(row, strconv.Itoa(eveningScore))
		} else {
			row = append(row, "")
		}

		if err := w.Write(row); err != nil {
			return nil, err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// generatePDF generates a PDF report in Russian with Cyrillic support
func (g *Generator) generatePDF(req CreateReportRequest, dailyMetrics []storage.DailyMetricRow, checkins []Checkin) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")

	// Try to add DejaVuSans font for Cyrillic support
	// Skip font loading in tests to avoid path issues
	fontName := "Arial" // Default fallback
	skipCustomFont := os.Getenv("SKIP_CUSTOM_FONT") == "1"

	if !skipCustomFont {
		// Use embedded font
		if len(embeddedFont) > 0 {
			// Create temporary file for font
			tmpFile, err := os.CreateTemp("", "DejaVuSans-*.ttf")
			if err == nil {
				defer os.Remove(tmpFile.Name())
				defer tmpFile.Close()

				// Write embedded font to temp file
				if _, err := tmpFile.Write(embeddedFont); err == nil {
					tmpFile.Close() // Close before PDF lib reads it

					// Try to add font, handle errors gracefully
					defer func() {
						if r := recover(); r != nil {
							// Font loading failed, use Arial
							fontName = "Arial"
						}
					}()

					pdf.AddUTF8Font("DejaVuSans", "", tmpFile.Name())
					fontName = "DejaVuSans"
				}
			}
		}
	}

	pdf.SetFont(fontName, "", 16)

	pdf.AddPage()

	// Title (transliterated if Arial fallback)
	if fontName == "DejaVuSans" {
		pdf.Cell(0, 10, "Отчёт о здоровье")
	} else {
		pdf.Cell(0, 10, "Health Report") // Fallback for tests
	}
	pdf.Ln(8)

	// Period
	pdf.SetFont(fontName, "", 12)
	pdf.Cell(0, 8, fmt.Sprintf("Период: %s — %s", req.From, req.To))
	pdf.Ln(12)

	// Calculate summary stats
	summary := g.calculateSummary(dailyMetrics, checkins)

	// Summary section
	pdf.SetFont(fontName, "", 14)
	pdf.Cell(0, 8, "Сводка")
	pdf.Ln(8)

	pdf.SetFont(fontName, "", 10)
	pdf.Cell(0, 6, fmt.Sprintf("Среднее количество шагов: %s", formatInt(summary.AvgSteps)))
	pdf.Ln(5)
	pdf.Cell(0, 6, fmt.Sprintf("Изменение веса: %s", summary.WeightDelta))
	pdf.Ln(5)
	pdf.Cell(0, 6, fmt.Sprintf("Средний пульс покоя: %s", formatInt(summary.AvgRestingHR)))
	pdf.Ln(5)
	pdf.Cell(0, 6, fmt.Sprintf("Средняя длительность сна: %s", summary.AvgSleep))
	pdf.Ln(5)
	pdf.Cell(0, 6, fmt.Sprintf("Средняя оценка (утро): %s", formatFloat(summary.AvgMorningScore)))
	pdf.Ln(5)
	pdf.Cell(0, 6, fmt.Sprintf("Средняя оценка (вечер): %s", formatFloat(summary.AvgEveningScore)))
	pdf.Ln(12)

	// Recent days table (last 14 days)
	pdf.SetFont(fontName, "", 14)
	pdf.Cell(0, 8, "Последние дни")
	pdf.Ln(8)

	g.drawRecentDaysTable(pdf, dailyMetrics, checkins, fontName)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// Summary holds calculated summary statistics
type Summary struct {
	AvgSteps        *int
	WeightDelta     string
	AvgRestingHR    *int
	AvgSleep        string
	AvgMorningScore *float64
	AvgEveningScore *float64
}

// calculateSummary calculates summary statistics
func (g *Generator) calculateSummary(dailyMetrics []storage.DailyMetricRow, checkins []Checkin) Summary {
	var totalSteps, countSteps int
	var totalHR, countHR int
	var totalSleep, countSleep int
	var firstWeight, lastWeight *float64
	var totalMorningScore, countMorning int
	var totalEveningScore, countEvening int

	// Process daily metrics
	for _, dm := range dailyMetrics {
		var payload map[string]interface{}
		if err := json.Unmarshal(dm.Payload, &payload); err != nil {
			continue
		}

		// Steps
		if activity, ok := payload["activity"].(map[string]interface{}); ok {
			if steps, ok := activity["steps"].(float64); ok {
				totalSteps += int(steps)
				countSteps++
			}
		}

		// Weight
		if body, ok := payload["body"].(map[string]interface{}); ok {
			if weight, ok := body["weight_kg_last"].(float64); ok {
				if firstWeight == nil {
					firstWeight = &weight
				}
				lastWeight = &weight
			}
		}

		// HR
		if heart, ok := payload["heart"].(map[string]interface{}); ok {
			if hr, ok := heart["resting_hr_bpm"].(float64); ok {
				totalHR += int(hr)
				countHR++
			}
		}

		// Sleep
		if sleep, ok := payload["sleep"].(map[string]interface{}); ok {
			if totalMin, ok := sleep["total_minutes"].(float64); ok {
				totalSleep += int(totalMin)
				countSleep++
			}
		}
	}

	// Process checkins
	for _, c := range checkins {
		if c.Type == "morning" {
			totalMorningScore += c.Score
			countMorning++
		} else if c.Type == "evening" {
			totalEveningScore += c.Score
			countEvening++
		}
	}

	summary := Summary{}

	if countSteps > 0 {
		avg := totalSteps / countSteps
		summary.AvgSteps = &avg
	}

	if firstWeight != nil && lastWeight != nil {
		delta := *lastWeight - *firstWeight
		summary.WeightDelta = fmt.Sprintf("%.1f кг", delta)
	} else {
		summary.WeightDelta = "Нет данных"
	}

	if countHR > 0 {
		avg := totalHR / countHR
		summary.AvgRestingHR = &avg
	}

	if countSleep > 0 {
		avgMin := totalSleep / countSleep
		hours := avgMin / 60
		minutes := avgMin % 60
		summary.AvgSleep = fmt.Sprintf("%dч %dм", hours, minutes)
	} else {
		summary.AvgSleep = "Нет данных"
	}

	if countMorning > 0 {
		avg := float64(totalMorningScore) / float64(countMorning)
		summary.AvgMorningScore = &avg
	}

	if countEvening > 0 {
		avg := float64(totalEveningScore) / float64(countEvening)
		summary.AvgEveningScore = &avg
	}

	return summary
}

// drawRecentDaysTable draws a table of recent days
func (g *Generator) drawRecentDaysTable(pdf *gofpdf.Fpdf, dailyMetrics []storage.DailyMetricRow, checkins []Checkin, fontName string) {
	// Limit to last 14 days
	limit := 14
	if len(dailyMetrics) < limit {
		limit = len(dailyMetrics)
	}

	recentMetrics := dailyMetrics
	if len(dailyMetrics) > limit {
		recentMetrics = dailyMetrics[len(dailyMetrics)-limit:]
	}

	// Create checkins map
	checkinsByDate := make(map[string]map[string]int)
	for _, c := range checkins {
		if checkinsByDate[c.Date] == nil {
			checkinsByDate[c.Date] = make(map[string]int)
		}
		checkinsByDate[c.Date][c.Type] = c.Score
	}

	pdf.SetFont(fontName, "", 8)

	// Table header
	pdf.CellFormat(25, 6, "Дата", "1", 0, "C", false, 0, "")
	pdf.CellFormat(20, 6, "Шаги", "1", 0, "C", false, 0, "")
	pdf.CellFormat(20, 6, "Вес", "1", 0, "C", false, 0, "")
	pdf.CellFormat(20, 6, "Пульс", "1", 0, "C", false, 0, "")
	pdf.CellFormat(20, 6, "Сон", "1", 0, "C", false, 0, "")
	pdf.CellFormat(20, 6, "Утро", "1", 0, "C", false, 0, "")
	pdf.CellFormat(20, 6, "Вечер", "1", 1, "C", false, 0, "")

	// Table rows
	for _, dm := range recentMetrics {
		var payload map[string]interface{}
		json.Unmarshal(dm.Payload, &payload)

		pdf.CellFormat(25, 6, dm.Date, "1", 0, "C", false, 0, "")

		// Steps
		steps := ""
		if activity, ok := payload["activity"].(map[string]interface{}); ok {
			if s, ok := activity["steps"].(float64); ok {
				steps = strconv.Itoa(int(s))
			}
		}
		pdf.CellFormat(20, 6, steps, "1", 0, "C", false, 0, "")

		// Weight
		weight := ""
		if body, ok := payload["body"].(map[string]interface{}); ok {
			if w, ok := body["weight_kg_last"].(float64); ok {
				weight = fmt.Sprintf("%.1f", w)
			}
		}
		pdf.CellFormat(20, 6, weight, "1", 0, "C", false, 0, "")

		// HR
		hr := ""
		if heart, ok := payload["heart"].(map[string]interface{}); ok {
			if h, ok := heart["resting_hr_bpm"].(float64); ok {
				hr = strconv.Itoa(int(h))
			}
		}
		pdf.CellFormat(20, 6, hr, "1", 0, "C", false, 0, "")

		// Sleep
		sleep := ""
		if sl, ok := payload["sleep"].(map[string]interface{}); ok {
			if totalMin, ok := sl["total_minutes"].(float64); ok {
				hours := int(totalMin) / 60
				sleep = fmt.Sprintf("%dч", hours)
			}
		}
		pdf.CellFormat(20, 6, sleep, "1", 0, "C", false, 0, "")

		// Morning score
		morning := ""
		if score, ok := checkinsByDate[dm.Date]["morning"]; ok {
			morning = strconv.Itoa(score)
		}
		pdf.CellFormat(20, 6, morning, "1", 0, "C", false, 0, "")

		// Evening score
		evening := ""
		if score, ok := checkinsByDate[dm.Date]["evening"]; ok {
			evening = strconv.Itoa(score)
		}
		pdf.CellFormat(20, 6, evening, "1", 1, "C", false, 0, "")
	}
}

// Helper functions
func formatInt(val *int) string {
	if val == nil {
		return "Нет данных"
	}
	return strconv.Itoa(*val)
}

func formatFloat(val *float64) string {
	if val == nil {
		return "Нет данных"
	}
	return fmt.Sprintf("%.1f", math.Round(*val*10)/10)
}

// CheckinsStorage interface for generator
type CheckinsStorage interface {
	ListCheckins(ctx context.Context, profileID uuid.UUID, from, to string) ([]Checkin, error)
}

// ProfileStorage interface for generator
type ProfileStorage interface {
	GetProfile(ctx context.Context, id uuid.UUID) (*storage.Profile, error)
}

// Checkin model for generator
type Checkin struct {
	ID        uuid.UUID
	ProfileID uuid.UUID
	Date      string
	Type      string
	Score     int
	Tags      []string
	Note      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
