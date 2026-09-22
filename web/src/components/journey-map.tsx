"use client";

import { GeoJSONSource, LngLatBounds, Map as MapLibreMap, NavigationControl } from "maplibre-gl";
import { useEffect, useMemo, useRef } from "react";
import type { LocalLocationSample, LocalStop, RouteSegment } from "@/lib/types";

const rasterStyle = {
  version: 8 as const,
  sources: {
    osm: {
      type: "raster" as const,
      tiles: ["https://tile.openstreetmap.org/{z}/{x}/{y}.png"],
      tileSize: 256,
      attribution: "© OpenStreetMap contributors",
    },
  },
  layers: [{ id: "osm", type: "raster" as const, source: "osm" }],
};

export function JourneyMap({
  samples,
  stops,
  route,
}: {
  samples: LocalLocationSample[];
  stops: LocalStop[];
  route: RouteSegment[];
}) {
  const container = useRef<HTMLDivElement>(null);
  const map = useRef<MapLibreMap | null>(null);
  const routeCoordinates = useMemo(() => {
    if (route.length) return route.flatMap((segment) => segment.geometry.coordinates);
    return samples.map((sample) => [sample.longitude, sample.latitude]);
  }, [route, samples]);

  useEffect(() => {
    if (!container.current || map.current) return;
    const instance = new MapLibreMap({
      container: container.current,
      style: rasterStyle,
      center: [77.209, 28.6139],
      zoom: 10,
      attributionControl: { compact: true },
    });
    map.current = instance;
    instance.addControl(new NavigationControl({ showCompass: false }), "top-right");
    return () => {
      map.current?.remove();
      map.current = null;
    };
  }, []);

  useEffect(() => {
    const current = map.current;
    if (!current) return;
    const update = () => {
      const routeData: GeoJSON.Feature<GeoJSON.LineString> = {
        type: "Feature",
        properties: {},
        geometry: { type: "LineString", coordinates: routeCoordinates },
      };
      const stopData: GeoJSON.FeatureCollection<GeoJSON.Point> = {
        type: "FeatureCollection",
        features: stops.map((stop) => ({
          type: "Feature",
          properties: { name: stop.name },
          geometry: { type: "Point", coordinates: [stop.longitude, stop.latitude] },
        })),
      };
      (current.getSource("journey-route") as GeoJSONSource | undefined)?.setData(routeData);
      (current.getSource("journey-stops") as GeoJSONSource | undefined)?.setData(stopData);
      if (!current.getSource("journey-route")) {
        current.addSource("journey-route", { type: "geojson", data: routeData });
        current.addLayer({
          id: "journey-route-line",
          type: "line",
          source: "journey-route",
          paint: { "line-color": "#e76f51", "line-width": 5, "line-opacity": 0.9 },
          layout: { "line-cap": "round", "line-join": "round" },
        });
      }
      if (!current.getSource("journey-stops")) {
        current.addSource("journey-stops", { type: "geojson", data: stopData });
        current.addLayer({
          id: "journey-stop-dots",
          type: "circle",
          source: "journey-stops",
          paint: {
            "circle-radius": 8,
            "circle-color": "#f4a261",
            "circle-stroke-color": "#123f35",
            "circle-stroke-width": 3,
          },
        });
      }

      const points = [...routeCoordinates, ...stops.map((stop) => [stop.longitude, stop.latitude])];
      if (points.length === 1) current.easeTo({ center: points[0] as [number, number], zoom: 15 });
      if (points.length > 1) {
        const bounds = points.reduce(
          (value, point) => value.extend(point as [number, number]),
          new LngLatBounds(points[0] as [number, number], points[0] as [number, number]),
        );
        current.fitBounds(bounds, { padding: 54, maxZoom: 16, duration: 500 });
      }
    };
    if (current.loaded()) update();
    else current.once("load", update);
  }, [routeCoordinates, stops]);

  return <div ref={container} className="journey-map" aria-label="Journey route map" />;
}
