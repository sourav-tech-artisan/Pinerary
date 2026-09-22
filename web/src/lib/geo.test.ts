import { describe, expect, it } from "vitest";
import { distanceMeters, formatDistance, formatDuration } from "@/lib/geo";

describe("geospatial presentation helpers", () => {
  it("computes a realistic distance between nearby Delhi coordinates", () => {
    const distance = distanceMeters(
      { latitude: 28.6129, longitude: 77.2295 },
      { latitude: 28.6562, longitude: 77.241 },
    );
    expect(distance).toBeGreaterThan(4_700);
    expect(distance).toBeLessThan(5_100);
  });

  it("formats distances and durations for compact cards", () => {
    expect(formatDistance(640)).toBe("640 m");
    expect(formatDistance(2_450)).toBe("2.5 km");
    expect(formatDuration(1_500)).toBe("25 min");
    expect(formatDuration(4_020)).toBe("1 hr 7 min");
  });
});
