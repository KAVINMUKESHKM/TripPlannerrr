export type User = {
  id: number;
  username?: string;
  email: string;
  full_name?: string;
};

export type Attraction = {
  id: number;
  name: string;
  description?: string;
  category?: string;
  best_climate?: string;
  ideal_weather?: string;
  ideal_temp_min?: number;
  ideal_temp_max?: number;
  latitude: number | string;
  longitude: number | string;
  source_url?: string;
  address?: string;
  opening_hours?: string;
  wheelchair?: string;
  website?: string;
  phone?: string;
  fee?: string;
  toilets?: string;
  toilets_wheelchair?: string;
  internet_access?: string;
  wikipedia?: string;
  wikidata?: string;
  details_updated_at?: string;
  image?: string;
  rating?: number;
};

export type NearbyRestaurant = {
  name: string;
  address?: string;
  rating: number;
  phone?: string;
  latitude: number;
  longitude: number;
  distanceKm: number;
};

export type DayPlan = {
  dayNumber: number;
  stops: Array<{ position: number; attractionId: number }>;
};

export type Trip = {
  id: number;
  destinationId?: number;
  destination: string;
  startDate: string;
  dayCount: number;
  days: DayPlan[];
  createdAt?: string;
  updatedAt?: string;
};

export type TripInput = {
  destination: string;
  startDate: string;
  dayCount: number;
  days: Array<{ dayNumber: number; attractionIds: number[] }>;
};

export type RouteSummary = {
  distanceMeters: number;
  durationSeconds: number;
  geometry: {
    type: "LineString";
    coordinates: [number, number][];
  };
};

export type SearchDay = {
  day: number;
  date?: string;
  weather?: string;
  temperature?: number;
  attractions: Attraction[];
};
