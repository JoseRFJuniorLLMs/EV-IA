package domain

import (
	"time"
)

// TripType represents the type of trip
type TripType string

const (
	TripTypeRoadTrip TripType = "road_trip"
	TripTypeFlight   TripType = "flight"
	TripTypeMixed    TripType = "mixed"
)

// Trip represents a planned trip with EV charging
type Trip struct {
	ID            string        `json:"id" gorm:"primaryKey"`
	UserID        string        `json:"user_id" gorm:"index"`
	Name          string        `json:"name"`
	Type          TripType      `json:"type"`
	Origin        Location      `json:"origin" gorm:"embedded;embeddedPrefix:origin_"`
	Destination   Location      `json:"destination" gorm:"embedded;embeddedPrefix:dest_"`
	DepartureDate time.Time     `json:"departure_date"`
	ReturnDate    *time.Time    `json:"return_date,omitempty"`
	Waypoints     []Waypoint    `json:"waypoints,omitempty" gorm:"serializer:json"`
	Flights       []Flight      `json:"flights,omitempty" gorm:"serializer:json"`
	Accommodations []Accommodation `json:"accommodations,omitempty" gorm:"serializer:json"`
	ChargingStops []ChargingStop `json:"charging_stops,omitempty" gorm:"serializer:json"`
	TotalDistance float64       `json:"total_distance_km"`
	EstimatedCost float64       `json:"estimated_cost"`
	Status        string        `json:"status"` // draft, planned, active, completed
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// Location represents a geographic location
type Location struct {
	Name      string  `json:"name"`
	Address   string  `json:"address,omitempty"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	PlaceID   string  `json:"place_id,omitempty"` // Google Places ID
}

// Waypoint represents a stop during a road trip
type Waypoint struct {
	Order       int       `json:"order"`
	Location    Location  `json:"location"`
	ArrivalTime time.Time `json:"arrival_time"`
	Duration    int       `json:"duration_minutes"` // How long to stay
	Type        string    `json:"type"`             // charging, rest, attraction, accommodation
	Notes       string    `json:"notes,omitempty"`
}

// ChargingStop represents a planned charging stop
type ChargingStop struct {
	Order           int       `json:"order"`
	ChargePointID   string    `json:"charge_point_id,omitempty"`
	Location        Location  `json:"location"`
	ArrivalTime     time.Time `json:"arrival_time"`
	EstimatedSOC    float64   `json:"estimated_soc_percent"`      // State of charge on arrival
	TargetSOC       float64   `json:"target_soc_percent"`         // Target state of charge
	ChargingTime    int       `json:"charging_time_minutes"`
	EstimatedCost   float64   `json:"estimated_cost"`
	ChargerType     string    `json:"charger_type"`               // DC Fast, Level 2, etc
	ChargerPower    float64   `json:"charger_power_kw"`
	ReservationID   string    `json:"reservation_id,omitempty"`
}

// Flight represents a flight booking
type Flight struct {
	ID              string    `json:"id"`
	Provider        string    `json:"provider"` // google_flights, skyscanner, etc
	ProviderID      string    `json:"provider_id"`
	Airline         string    `json:"airline"`
	FlightNumber    string    `json:"flight_number"`
	Origin          Airport   `json:"origin"`
	Destination     Airport   `json:"destination"`
	DepartureTime   time.Time `json:"departure_time"`
	ArrivalTime     time.Time `json:"arrival_time"`
	Duration        int       `json:"duration_minutes"`
	Class           string    `json:"class"` // economy, business, first
	Price           float64   `json:"price"`
	Currency        string    `json:"currency"`
	BookingURL      string    `json:"booking_url,omitempty"`
	Status          string    `json:"status"` // available, booked, cancelled
	Layovers        []Layover `json:"layovers,omitempty"`
	BaggageIncluded bool      `json:"baggage_included"`
	Refundable      bool      `json:"refundable"`
}

// Airport represents an airport
type Airport struct {
	Code    string  `json:"code"` // IATA code
	Name    string  `json:"name"`
	City    string  `json:"city"`
	Country string  `json:"country"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

// Layover represents a flight layover
type Layover struct {
	Airport  Airport `json:"airport"`
	Duration int     `json:"duration_minutes"`
}

// Accommodation represents a hotel/accommodation booking
type Accommodation struct {
	ID                string    `json:"id"`
	Provider          string    `json:"provider"` // airbnb, booking, hotels
	ProviderID        string    `json:"provider_id"`
	Name              string    `json:"name"`
	Type              string    `json:"type"` // hotel, apartment, house, hostel
	Location          Location  `json:"location"`
	CheckIn           time.Time `json:"check_in"`
	CheckOut          time.Time `json:"check_out"`
	Nights            int       `json:"nights"`
	Guests            int       `json:"guests"`
	Rooms             int       `json:"rooms"`
	PricePerNight     float64   `json:"price_per_night"`
	TotalPrice        float64   `json:"total_price"`
	Currency          string    `json:"currency"`
	Rating            float64   `json:"rating"`
	ReviewCount       int       `json:"review_count"`
	Amenities         []string  `json:"amenities"`
	HasEVCharging     bool      `json:"has_ev_charging"`
	EVChargingDetails string    `json:"ev_charging_details,omitempty"`
	Images            []string  `json:"images,omitempty"`
	BookingURL        string    `json:"booking_url"`
	CancellationPolicy string   `json:"cancellation_policy"`
	Status            string    `json:"status"` // available, booked, cancelled
}

// FlightSearchRequest represents a flight search request
type FlightSearchRequest struct {
	Origin        string    `json:"origin"`        // IATA code or city
	Destination   string    `json:"destination"`   // IATA code or city
	DepartureDate time.Time `json:"departure_date"`
	ReturnDate    *time.Time `json:"return_date,omitempty"`
	Passengers    int       `json:"passengers"`
	Class         string    `json:"class"`
	MaxStops      int       `json:"max_stops"`
	MaxPrice      float64   `json:"max_price,omitempty"`
	Currency      string    `json:"currency"`
}

// AccommodationSearchRequest represents an accommodation search request
type AccommodationSearchRequest struct {
	Location      string    `json:"location"` // City or coordinates
	Latitude      float64   `json:"latitude,omitempty"`
	Longitude     float64   `json:"longitude,omitempty"`
	Radius        float64   `json:"radius_km,omitempty"`
	CheckIn       time.Time `json:"check_in"`
	CheckOut      time.Time `json:"check_out"`
	Guests        int       `json:"guests"`
	Rooms         int       `json:"rooms"`
	MinPrice      float64   `json:"min_price,omitempty"`
	MaxPrice      float64   `json:"max_price,omitempty"`
	Currency      string    `json:"currency"`
	Type          string    `json:"type,omitempty"` // hotel, apartment, any
	MinRating     float64   `json:"min_rating,omitempty"`
	RequireEVCharging bool  `json:"require_ev_charging"`
	Amenities     []string  `json:"amenities,omitempty"`
}

// TripPlanRequest represents a trip planning request
type TripPlanRequest struct {
	UserID          string     `json:"user_id"`
	Origin          Location   `json:"origin"`
	Destination     Location   `json:"destination"`
	DepartureDate   time.Time  `json:"departure_date"`
	ReturnDate      *time.Time `json:"return_date,omitempty"`
	TripType        TripType   `json:"trip_type"`
	VehicleRange    float64    `json:"vehicle_range_km"`    // EV range on full charge
	CurrentSOC      float64    `json:"current_soc_percent"` // Current battery level
	PreferredSOC    float64    `json:"preferred_soc_percent"` // Preferred minimum SOC
	Passengers      int        `json:"passengers"`
	NeedAccommodation bool     `json:"need_accommodation"`
	AccommodationType string   `json:"accommodation_type,omitempty"`
	MaxBudget       float64    `json:"max_budget,omitempty"`
	Currency        string     `json:"currency"`
	Preferences     TripPreferences `json:"preferences"`
}

// TripPreferences represents user preferences for trip planning
type TripPreferences struct {
	PreferFastCharging  bool     `json:"prefer_fast_charging"`
	AvoidTolls          bool     `json:"avoid_tolls"`
	AvoidHighways       bool     `json:"avoid_highways"`
	MaxDrivingHours     float64  `json:"max_driving_hours_per_day"`
	PreferredChargers   []string `json:"preferred_chargers,omitempty"` // Brand preferences
	RequireAmenities    []string `json:"require_amenities,omitempty"`  // Restroom, food, wifi
	PreferEVCharging    bool     `json:"prefer_ev_charging_accommodation"`
}

// TripEstimate provides cost and time estimates
type TripEstimate struct {
	TotalDistance       float64 `json:"total_distance_km"`
	TotalDuration       int     `json:"total_duration_minutes"`
	DrivingDuration     int     `json:"driving_duration_minutes"`
	ChargingDuration    int     `json:"charging_duration_minutes"`
	ChargingStops       int     `json:"charging_stops"`
	EstimatedChargingCost float64 `json:"estimated_charging_cost"`
	EstimatedFlightCost   float64 `json:"estimated_flight_cost,omitempty"`
	EstimatedAccommodationCost float64 `json:"estimated_accommodation_cost,omitempty"`
	TotalEstimatedCost  float64 `json:"total_estimated_cost"`
	Currency            string  `json:"currency"`
}
