"use client";

import "maplibre-gl/dist/maplibre-gl.css";
import { useEffect, useRef, useState } from "react";
import { LngLatBounds, Map as MapLibreMap, Marker as MapMarker, NavigationControl, Popup, setWorkerUrl, type GeoJSONSource } from "maplibre-gl";
import type { Attraction, RouteSummary } from "@/lib/types";

const DEFAULT_STYLE = "https://tiles.openfreemap.org/styles/liberty";
const FALLBACK_STYLE = "https://tiles.openfreemap.org/styles/positron";
setWorkerUrl(new URL("maplibre-gl/dist/maplibre-gl-worker.mjs", import.meta.url).toString());

export default function MapLibreMapView({ places, route, focusPlace }: { places: Attraction[]; route: RouteSummary | null; focusPlace: Attraction | null }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<MapLibreMap | null>(null);
  const markersRef = useRef<MapMarker[]>([]);
  const routeRef = useRef<RouteSummary | null>(route);
  const placesRef = useRef<Attraction[]>(places);
  const fallbackAppliedRef = useRef(false);
  const mapReadyRef = useRef(false);
  const [mapError, setMapError] = useState("");
  const styleURL = process.env.NEXT_PUBLIC_OPENFREEMAP_STYLE_URL || DEFAULT_STYLE;
  routeRef.current = route;
  placesRef.current = places;

  useEffect(() => {
    if (!containerRef.current) return;
    const map = new MapLibreMap({
      container: containerRef.current,
      style: styleURL,
      center: focusPlace ? [Number(focusPlace.longitude), Number(focusPlace.latitude)] : places.length ? [Number(places[0].longitude), Number(places[0].latitude)] : [77.5946, 12.9716],
      zoom: focusPlace || places.length ? 13 : 3,
    });
    mapRef.current = map;
    let errorTimer: ReturnType<typeof setTimeout> | undefined;
    map.addControl(new NavigationControl({ showCompass: false }), "top-right");
    fallbackAppliedRef.current = false;
    mapReadyRef.current = false;
    map.on("error", () => {
      if (mapReadyRef.current) return;
      if (!fallbackAppliedRef.current) {
        fallbackAppliedRef.current = true;
        map.setStyle(styleURL === DEFAULT_STYLE ? FALLBACK_STYLE : DEFAULT_STYLE);
        return;
      }
      if (!errorTimer) errorTimer = setTimeout(() => {
        if (!mapReadyRef.current) setMapError("The map style could not load. Check your connection to OpenFreeMap.");
      }, 10000);
    });

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
          current.addLayer({ id: "trip-route-line", type: "line", source: "trip-route", paint: { "line-color": "#c6ac8f", "line-width": 4, "line-opacity": 0.96 }, layout: { "line-cap": "round", "line-join": "round" } });
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
    map.on("load", () => { mapReadyRef.current = true; setMapError(""); if (errorTimer) clearTimeout(errorTimer); drawRoute(); });
    map.on("style.load", drawRoute);
    return () => { if (errorTimer) clearTimeout(errorTimer); markersRef.current.forEach((marker) => marker.remove()); markersRef.current = []; map.remove(); mapRef.current = null; };
  }, [styleURL]);

  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;
    markersRef.current.forEach((marker) => marker.remove());
    const visiblePlaces = [...places];
    if (focusPlace && !visiblePlaces.some((place) => place.id === focusPlace.id)) visiblePlaces.push(focusPlace);
    const validPlaces = visiblePlaces.filter((place) => Number.isFinite(Number(place.latitude)) && Number.isFinite(Number(place.longitude)) && Number(place.latitude) !== 0 && Number(place.longitude) !== 0);
    markersRef.current = validPlaces.map((place) => {
      const isFocused = focusPlace?.id === place.id;
      const marker = new MapMarker({ color: isFocused ? "#c6ac8f" : "#22333b" }).setLngLat([Number(place.longitude), Number(place.latitude)]).setPopup(new Popup({ offset: 18 }).setText(place.name)).addTo(map);
      return marker;
    });
    if (focusPlace && Number.isFinite(Number(focusPlace.latitude)) && Number.isFinite(Number(focusPlace.longitude))) {
      map.flyTo({ center: [Number(focusPlace.longitude), Number(focusPlace.latitude)], zoom: 15, duration: 700 });
    } else if (validPlaces.length === 1) {
      map.flyTo({ center: [Number(validPlaces[0].longitude), Number(validPlaces[0].latitude)], zoom: 15, duration: 700 });
    } else if (validPlaces.length > 1) {
      const first = validPlaces[0];
      const bounds = validPlaces.slice(1).reduce((value, place) => value.extend([Number(place.longitude), Number(place.latitude)]), new LngLatBounds([Number(first.longitude), Number(first.latitude)], [Number(first.longitude), Number(first.latitude)]));
      map.fitBounds(bounds, { padding: 64, maxZoom: 14, duration: 700 });
    }
  }, [places, focusPlace]);

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
          map.addLayer({ id: "trip-route-line", type: "line", source: "trip-route", paint: { "line-color": "#c6ac8f", "line-width": 4, "line-opacity": 0.96 }, layout: { "line-cap": "round", "line-join": "round" } });
        }
        const coordinates = route.geometry.coordinates;
        if (coordinates.length === 1) {
          map.flyTo({ center: coordinates[0], zoom: 15, duration: 700 });
        } else if (coordinates.length > 1) {
          const bounds = coordinates.slice(1).reduce((value, point) => value.extend(point), new LngLatBounds(coordinates[0], coordinates[0]));
          map.fitBounds(bounds, { padding: 64, maxZoom: 14, duration: 700 });
        }
      } else if (source) source.setData({ type: "FeatureCollection", features: [] });
    };
    if (map.isStyleLoaded()) update(); else map.once("load", update);
    return () => { map.off("load", update); };
  }, [route]);

  return <div className="map-frame"><div className="map-canvas" ref={containerRef} role="region" aria-label="Map showing your selected trip stops" />{mapError ? <p className="map-error" role="status">{mapError}</p> : null}</div>;
}
