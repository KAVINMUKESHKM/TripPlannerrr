"use client";

import "maplibre-gl/dist/maplibre-gl.css";
import { useEffect, useRef, useState } from "react";
import { LngLatBounds, Map as MapLibreMap, Marker as MapMarker, NavigationControl, Popup, type GeoJSONSource } from "maplibre-gl";
import type { Attraction, RouteSummary } from "@/lib/types";

const DEFAULT_STYLE = "https://tiles.openfreemap.org/styles/liberty";

export default function MapLibreMapView({ places, route }: { places: Attraction[]; route: RouteSummary | null }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<MapLibreMap | null>(null);
  const markersRef = useRef<MapMarker[]>([]);
  const routeRef = useRef<RouteSummary | null>(route);
  const placesRef = useRef<Attraction[]>(places);
  const [mapError, setMapError] = useState("");
  const styleURL = process.env.NEXT_PUBLIC_OPENFREEMAP_STYLE_URL || DEFAULT_STYLE;
  routeRef.current = route;
  placesRef.current = places;

  useEffect(() => {
    if (!containerRef.current) return;
    const map = new MapLibreMap({
      container: containerRef.current,
      style: styleURL,
      center: places.length ? [Number(places[0].longitude), Number(places[0].latitude)] : [12.9716, 77.5946],
      zoom: places.length ? 11 : 3,
    });
    mapRef.current = map;
    map.addControl(new NavigationControl({ showCompass: false }), "top-right");
    map.on("error", () => setMapError("The map tiles could not be loaded. Check the configured OpenFreeMap style URL."));

    const drawRoute = () => {
      const current = mapRef.current;
      if (!current || !current.isStyleLoaded()) return;
      const geometry = routeRef.current?.geometry;
      const source = current.getSource("trip-route") as GeoJSONSource | undefined;
      if (geometry) {
        const feature = { type: "Feature" as const, properties: {}, geometry };
        if (source) source.setData(feature);
        else {
          current.addSource("trip-route", { type: "geojson", data: feature });
          current.addLayer({ id: "trip-route-line", type: "line", source: "trip-route", paint: { "line-color": "#c67c36", "line-width": 4, "line-opacity": 0.88 }, layout: { "line-cap": "round", "line-join": "round" } });
        }
      } else if (source) source.setData({ type: "FeatureCollection", features: [] });
      const points = placesRef.current.filter((place) => Number.isFinite(Number(place.latitude)) && Number.isFinite(Number(place.longitude)) && Number(place.latitude) !== 0 && Number(place.longitude) !== 0);
      const routePoints = geometry?.coordinates?.map(([longitude, latitude]) => ({ longitude, latitude })) ?? [];
      const coordinates = routePoints.length ? routePoints : points.map((place) => ({ longitude: Number(place.longitude), latitude: Number(place.latitude) }));
      if (coordinates.length === 1) current.flyTo({ center: [coordinates[0].longitude, coordinates[0].latitude], zoom: 13, duration: 500 });
      else if (coordinates.length > 1) {
        const bounds = coordinates.reduce((value, point) => value.extend([point.longitude, point.latitude]), new LngLatBounds([coordinates[0].longitude, coordinates[0].latitude], [coordinates[0].longitude, coordinates[0].latitude]));
        current.fitBounds(bounds, { padding: 56, maxZoom: 14, duration: 500 });
      }
    };
    map.on("load", drawRoute);
    return () => { markersRef.current.forEach((marker) => marker.remove()); markersRef.current = []; map.remove(); mapRef.current = null; };
  }, [styleURL]);

  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;
    markersRef.current.forEach((marker) => marker.remove());
    markersRef.current = places.filter((place) => Number.isFinite(Number(place.latitude)) && Number.isFinite(Number(place.longitude)) && Number(place.latitude) !== 0 && Number(place.longitude) !== 0).map((place, index) => {
      const marker = new MapMarker({ color: index === 0 ? "#c67c36" : "#173c35" }).setLngLat([Number(place.longitude), Number(place.latitude)]).setPopup(new Popup({ offset: 18 }).setText(place.name)).addTo(map);
      return marker;
    });
  }, [places]);

  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;
    const update = () => {
      if (!map.isStyleLoaded()) return;
      const source = map.getSource("trip-route") as GeoJSONSource | undefined;
      if (route?.geometry) {
        const feature = { type: "Feature" as const, properties: {}, geometry: route.geometry };
        if (source) source.setData(feature);
        else {
          map.addSource("trip-route", { type: "geojson", data: feature });
          map.addLayer({ id: "trip-route-line", type: "line", source: "trip-route", paint: { "line-color": "#c67c36", "line-width": 4, "line-opacity": 0.88 }, layout: { "line-cap": "round", "line-join": "round" } });
        }
      } else if (source) source.setData({ type: "FeatureCollection", features: [] });
    };
    if (map.isStyleLoaded()) update(); else map.once("load", update);
    return () => { map.off("load", update); };
  }, [route]);

  return <div className="map-frame"><div className="map-canvas" ref={containerRef} role="region" aria-label="Map showing your selected trip stops" />{mapError ? <p className="map-error" role="status">{mapError}</p> : null}</div>;
}
