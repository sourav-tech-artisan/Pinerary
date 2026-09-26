"use client";

import { Map as MapLibreMap, Marker, NavigationControl } from "maplibre-gl";
import { useEffect, useRef } from "react";
import type { Coordinates } from "@/lib/types";

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

export function PlacePreviewMap({ coordinates, name }: { coordinates: Coordinates; name: string }) {
  const container = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!container.current) return;
    const center: [number, number] = [coordinates.longitude, coordinates.latitude];
    const map = new MapLibreMap({
      container: container.current,
      style: rasterStyle,
      center,
      zoom: 15,
      attributionControl: { compact: true },
    });
    map.addControl(new NavigationControl({ showCompass: false }), "top-right");
    new Marker({ color: "#e76f51" }).setLngLat(center).addTo(map);
    return () => map.remove();
  }, [coordinates.latitude, coordinates.longitude]);

  return <div ref={container} className="place-preview-map" role="img" aria-label={`Map showing ${name || "selected place"}`} />;
}
