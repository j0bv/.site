package selectrucks

import (
	"testing"
)

func TestConvertTruckToExtract_NilWhenTruckIdZero(t *testing.T) {
	s := NewSelecTrucksScraper()
	truck := &Truck{
		Manufacturer: "Peterbilt",
		Model:        "389",
		Year:         2022,
		TruckId:      0,
	}
	got := s.ConvertTruckToExtract(truck)
	if got != nil {
		t.Fatalf("ConvertTruckToExtract with TruckId 0 should return nil, got %v", got)
	}
}

func TestConvertTruckToExtract_NilWhenTruckIdNegative(t *testing.T) {
	s := NewSelecTrucksScraper()
	truck := &Truck{TruckId: -1}
	got := s.ConvertTruckToExtract(truck)
	if got != nil {
		t.Fatalf("ConvertTruckToExtract with TruckId -1 should return nil, got %v", got)
	}
}

func TestConvertTruckToExtract_Minimal(t *testing.T) {
	s := NewSelecTrucksScraper()
	truck := &Truck{TruckId: 1001}
	got := s.ConvertTruckToExtract(truck)
	if got == nil {
		t.Fatal("ConvertTruckToExtract with TruckId should not return nil")
	}
	if got.URL != "https://www.selectrucks.com/trucks/1001" {
		t.Errorf("URL = %q, want https://www.selectrucks.com/trucks/1001", got.URL)
	}
}

func TestConvertTruckToExtract_Full(t *testing.T) {
	s := NewSelecTrucksScraper()
	truck := &Truck{
		TruckId:                   2002,
		Manufacturer:              "Volvo",
		Model:                     "VNL",
		Year:                      2020,
		Price:                     "$75,000.00",
		Mileage:                   "150,000",
		StockNumber:               "STK789",
		ImageFileName:              "volvo-vnl-2002.jpg",
		DealerName:                "Test Dealer",
		DealerCountryAbbreviation: "US",
	}
	got := s.ConvertTruckToExtract(truck)
	if got == nil {
		t.Fatal("ConvertTruckToExtract should not return nil")
	}
	if got.URL != "https://www.selectrucks.com/trucks/2002" {
		t.Errorf("URL = %q", got.URL)
	}
	if got.Make == nil || *got.Make != "Volvo" {
		t.Errorf("Make = %v", got.Make)
	}
	if got.Model == nil || *got.Model != "VNL" {
		t.Errorf("Model = %v", got.Model)
	}
	if got.Year == nil || *got.Year != 2020 {
		t.Errorf("Year = %v", got.Year)
	}
	if got.Price == nil || *got.Price != 75000 {
		t.Errorf("Price = %v (parsed from $75,000.00)", got.Price)
	}
	if got.Miles == nil || *got.Miles != 150000 {
		t.Errorf("Miles = %v (parsed from 150,000)", got.Miles)
	}
	if got.StockNumber == nil || *got.StockNumber != "STK789" {
		t.Errorf("StockNumber = %v", got.StockNumber)
	}
	if got.ImageURL == nil || *got.ImageURL != "https://www.selectrucks.com/images/trucks/volvo-vnl-2002.jpg" {
		t.Errorf("ImageURL = %v", got.ImageURL)
	}
	if got.Location == nil || *got.Location != "Test Dealer, US" {
		t.Errorf("Location = %v", got.Location)
	}
	if got.Title == nil || *got.Title != "2020 Volvo VNL" {
		t.Errorf("Title = %v", got.Title)
	}
}
