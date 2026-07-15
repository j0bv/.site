package arrowtruck

import (
	"testing"
)

func TestConvertProductToExtract_NilWhenNoStockNumber(t *testing.T) {
	s := NewArrowTruckScraper()
	product := &Product{
		Make: "Peterbilt",
		Model: "389",
		Year: 2022,
		StockNumber: "",
	}
	got := s.ConvertProductToExtract(product)
	if got != nil {
		t.Fatalf("ConvertProductToExtract with empty StockNumber should return nil, got %v", got)
	}
}

func TestConvertProductToExtract_Minimal(t *testing.T) {
	s := NewArrowTruckScraper()
	product := &Product{
		StockNumber: "STK123",
	}
	got := s.ConvertProductToExtract(product)
	if got == nil {
		t.Fatal("ConvertProductToExtract with StockNumber should not return nil")
	}
	if got.URL != "https://www.arrowtruck.com/trucks/STK123" {
		t.Errorf("URL = %q, want https://www.arrowtruck.com/trucks/STK123", got.URL)
	}
	if got.StockNumber == nil || *got.StockNumber != "STK123" {
		t.Errorf("StockNumber = %v, want ptr to \"STK123\"", got.StockNumber)
	}
}

func TestConvertProductToExtract_Full(t *testing.T) {
	s := NewArrowTruckScraper()
	year := 2021
	product := &Product{
		StockNumber:   "STK456",
		Make:          "Freightliner",
		Model:         " Cascadia",
		Year:          year,
		LocationCity:  "Dallas",
		LocationState: "TX",
		Headline:      "Great condition",
		Description:   "Full service history",
		Price:         PriceValue{Value: 85000},
		Mileage:       MileageValue{Value: 250000},
		Media: []MediaProvider{
			{ProviderName: "Arrow", Media: MediaContent{Photos: []Photo{{MediaURL: "https://example.com/img.jpg"}}}},
			{ProviderName: "Glo3D", Media: MediaContent{VIN: "1NP5DB9X7MN567890"}},
		},
	}
	got := s.ConvertProductToExtract(product)
	if got == nil {
		t.Fatal("ConvertProductToExtract should not return nil")
	}
	if got.URL != "https://www.arrowtruck.com/trucks/STK456" {
		t.Errorf("URL = %q", got.URL)
	}
	if got.Make == nil || *got.Make != "Freightliner" {
		t.Errorf("Make = %v", got.Make)
	}
	if got.Model == nil || *got.Model != "Cascadia" {
		t.Errorf("Model = %v (trimmed)", got.Model)
	}
	if got.Year == nil || *got.Year != 2021 {
		t.Errorf("Year = %v", got.Year)
	}
	if got.Price == nil || *got.Price != 85000 {
		t.Errorf("Price = %v", got.Price)
	}
	if got.Miles == nil || *got.Miles != 250000 {
		t.Errorf("Miles = %v", got.Miles)
	}
	if got.Location == nil || *got.Location != "Dallas, TX" {
		t.Errorf("Location = %v", got.Location)
	}
	if got.ImageURL == nil || *got.ImageURL != "https://example.com/img.jpg" {
		t.Errorf("ImageURL = %v", got.ImageURL)
	}
	if got.VIN == nil || *got.VIN != "1NP5DB9X7MN567890" {
		t.Errorf("VIN = %v", got.VIN)
	}
	if got.Description == nil || *got.Description != "Full service history" {
		t.Errorf("Description = %v (prefer Description over Headline)", got.Description)
	}
	if got.Title == nil || *got.Title != "Great condition" {
		t.Errorf("Title = %v (Headline when set)", got.Title)
	}
}
