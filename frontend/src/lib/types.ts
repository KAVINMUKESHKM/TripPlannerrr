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
  latitude: number | string;
  longitude: number | string;
  image?: string;
  address?: string;
  rating?: number;
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
