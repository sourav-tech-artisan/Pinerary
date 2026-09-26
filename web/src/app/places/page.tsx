"use client";

import { useLiveQuery } from "dexie-react-hooks";
import { ExternalLink, LoaderCircle, MapPin, Plus, Search } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useApp } from "@/components/app-provider";
import { PlacePreviewMap } from "@/components/place-preview-map";
import { EmptyState, InlineMessage, PageHeading, SyncBadge } from "@/components/ui";
import { APIError, reverseGeocode, searchPlaces } from "@/lib/api/client";
import { getOwnerKey } from "@/lib/config";
import { db } from "@/lib/db";
import { currentCoordinates } from "@/lib/geo";
import { normalizePlaceQuery, PlaceSearchCoordinator, type PlaceSearchCallbacks } from "@/lib/place-search";
import { saveStandalonePlace } from "@/lib/repository";
import type { Coordinates, PlaceSearchResult, PlaceSuggestion } from "@/lib/types";

function searchErrorMessage(reason: unknown): string {
  if (typeof navigator !== "undefined" && !navigator.onLine) return "Connect to the internet to search for new places.";
  if (reason instanceof APIError) {
    if (reason.code === "place_search_quota_exhausted") return "Today’s place-search allowance has been used. Try again tomorrow.";
    if (reason.code === "rate_limited") return "You’re searching too quickly. Pause briefly, then try again.";
    if (reason.code === "place_search_unavailable") return "Place suggestions are temporarily unavailable. Try again shortly.";
    if (reason.code === "invalid_search_query") return "Enter at least three characters to search.";
  }
  return "Could not search for places. Check your connection and try again.";
}

export default function PlacesPage() {
  const ownerKey = getOwnerKey();
  const { syncNow } = useApp();
  const places = useLiveQuery(() => db.places.where("ownerKey").equals(ownerKey).toArray(), [ownerKey]);
  const [savedQuery, setSavedQuery] = useState("");
  const [discoveryQuery, setDiscoveryQuery] = useState("");
  const [resultQuery, setResultQuery] = useState("");
  const [searchResult, setSearchResult] = useState<PlaceSearchResult>();
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState("");
  const [suggestionsOpen, setSuggestionsOpen] = useState(false);
  const [activeSuggestion, setActiveSuggestion] = useState(-1);
  const [showForm, setShowForm] = useState(false);
  const [draftSource, setDraftSource] = useState<"current" | "search">("current");
  const [selectedAddress, setSelectedAddress] = useState("");
  const [coordinates, setCoordinates] = useState<Coordinates>();
  const [name, setName] = useState("");
  const [notes, setNotes] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const captureGeneration = useRef(0);
  const searchCoordinator = useMemo(
    () => new PlaceSearchCoordinator((query, signal) => {
      if (!navigator.onLine) return Promise.reject(new Error("offline"));
      return searchPlaces(query, 5, signal);
    }),
    [],
  );
  const searchCallbacks = useMemo<PlaceSearchCallbacks>(() => ({
    onReset: () => {
      setSearching(false);
      setSearchError("");
      setSearchResult(undefined);
      setResultQuery("");
      setSuggestionsOpen(false);
      setActiveSuggestion(-1);
    },
    onLoading: () => {
      setSearching(true);
      setSearchError("");
      setSuggestionsOpen(true);
      setActiveSuggestion(-1);
    },
    onSuccess: (query, result) => {
      setSearching(false);
      setSearchError("");
      setSearchResult(result);
      setResultQuery(query);
      setSuggestionsOpen(true);
      setActiveSuggestion(-1);
    },
    onError: (query, reason) => {
      setSearching(false);
      setSearchError(searchErrorMessage(reason));
      setSearchResult(undefined);
      setResultQuery(query);
      setSuggestionsOpen(true);
      setActiveSuggestion(-1);
    },
  }), []);
  const normalizedDiscoveryQuery = normalizePlaceQuery(discoveryQuery);
  const visibleResult = resultQuery === normalizedDiscoveryQuery ? searchResult : undefined;
  const suggestions = visibleResult?.items ?? [];
  const showSuggestions = suggestionsOpen && normalizedDiscoveryQuery.length >= 3;
  const filtered = useMemo(
    () => (places ?? [])
      .filter((place) => `${place.name} ${place.notes}`.toLowerCase().includes(savedQuery.toLowerCase()))
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt)),
    [places, savedQuery],
  );

  useEffect(() => {
    searchCoordinator.schedule(discoveryQuery, searchCallbacks);
    return () => searchCoordinator.cancel();
  }, [discoveryQuery, searchCallbacks, searchCoordinator]);

  useEffect(() => () => searchCoordinator.cancel(), [searchCoordinator]);

  function closeForm() {
    captureGeneration.current += 1;
    setShowForm(false);
    setBusy(false);
    setCoordinates(undefined);
    setSelectedAddress("");
    setName("");
    setNotes("");
    setError("");
  }

  async function beginCapture() {
    const generation = ++captureGeneration.current;
    setDraftSource("current");
    setSelectedAddress("");
    setCoordinates(undefined);
    setName("");
    setNotes("");
    setShowForm(true);
    setBusy(true);
    setError("");
    try {
      const location = await currentCoordinates();
      if (generation !== captureGeneration.current) return;
      setCoordinates(location);
      const suggestion = navigator.onLine
        ? await reverseGeocode(location.latitude, location.longitude).catch(() => undefined)
        : undefined;
      if (generation !== captureGeneration.current) return;
      setName(suggestion?.display_name || "Saved place");
    } catch (reason) {
      if (generation === captureGeneration.current) {
        setError(reason instanceof Error ? reason.message : "Could not capture your location.");
      }
    } finally {
      if (generation === captureGeneration.current) setBusy(false);
    }
  }

  function chooseSuggestion(suggestion: PlaceSuggestion) {
    captureGeneration.current += 1;
    searchCoordinator.cancel();
    setSearching(false);
    setSuggestionsOpen(false);
    setDraftSource("search");
    setSelectedAddress(suggestion.address);
    setCoordinates({ latitude: suggestion.latitude, longitude: suggestion.longitude });
    setName(suggestion.name);
    setNotes("");
    setError("");
    setBusy(false);
    setShowForm(true);
  }

  function submitSearch(event: React.FormEvent) {
    event.preventDefault();
    void searchCoordinator.runNow(discoveryQuery, searchCallbacks);
  }

  function handleSearchKeys(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Escape") {
      searchCoordinator.cancel();
      setSearching(false);
      setSuggestionsOpen(false);
      setActiveSuggestion(-1);
      return;
    }
    if (!suggestions.length) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setSuggestionsOpen(true);
      setActiveSuggestion((value) => Math.min(value + 1, suggestions.length - 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActiveSuggestion((value) => Math.max(value - 1, 0));
    } else if (event.key === "Enter" && activeSuggestion >= 0) {
      event.preventDefault();
      chooseSuggestion(suggestions[activeSuggestion]);
    }
  }

  async function save(event: React.FormEvent) {
    event.preventDefault();
    if (!coordinates || !name.trim()) return;
    setBusy(true);
    setError("");
    try {
      await saveStandalonePlace(name.trim(), notes.trim(), coordinates);
      closeForm();
      void syncNow();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not save this place.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="page">
      <PageHeading
        eyebrow="Private place library"
        title="Places worth returning to"
        description="Save somewhere now or collect places before a future trip. Every pin remains available offline."
        action={<button className="button primary" onClick={beginCapture}><Plus size={18} /> Pin current place</button>}
      />

      <section className="place-discovery" aria-labelledby="place-discovery-title">
        <div className="place-discovery-heading">
          <div>
            <p className="eyebrow">Plan ahead</p>
            <h2 id="place-discovery-title">Search for a new place</h2>
            <p id="place-search-help">Try a partial name, address, city, or landmark. Online search text is sent to Geoapify.</p>
          </div>
        </div>
        <form onSubmit={submitSearch} role="search">
          <div className="autocomplete">
            <label className="search-box discovery-search" htmlFor="place-discovery-input">
              <Search size={18} />
              <input
                id="place-discovery-input"
                value={discoveryQuery}
                onChange={(event) => {
                  setDiscoveryQuery(event.target.value);
                  setSearching(false);
                  setSearchError("");
                  setSuggestionsOpen(true);
                  setActiveSuggestion(-1);
                }}
                onFocus={() => setSuggestionsOpen(true)}
                onKeyDown={handleSearchKeys}
                placeholder="Search hotels, cafés, landmarks, or addresses"
                maxLength={200}
                autoComplete="off"
                role="combobox"
                aria-autocomplete="list"
                aria-expanded={showSuggestions}
                aria-controls="place-suggestions"
                aria-describedby="place-search-help"
                aria-activedescendant={activeSuggestion >= 0 ? `place-suggestion-${activeSuggestion}` : undefined}
              />
              {searching && <LoaderCircle className="spin" size={18} aria-label="Searching" />}
            </label>

            {showSuggestions && (
              <div className="suggestion-panel">
                {searchError && <InlineMessage tone="error">{searchError}</InlineMessage>}
                {!searching && !searchError && visibleResult && suggestions.length === 0 && (
                  <p className="suggestion-empty" role="status">No matching places. Add a city, area, or more detail and try again.</p>
                )}
                <ul id="place-suggestions" className="suggestion-list" role="listbox" aria-label="Place suggestions">
                  {suggestions.map((suggestion, index) => (
                    <li
                      id={`place-suggestion-${index}`}
                      key={suggestion.result_id}
                      role="option"
                      aria-selected={activeSuggestion === index}
                      className={activeSuggestion === index ? "active" : ""}
                    >
                      <button
                        type="button"
                        tabIndex={-1}
                        onMouseEnter={() => setActiveSuggestion(index)}
                        onClick={() => chooseSuggestion(suggestion)}
                      >
                        <span className="suggestion-pin"><MapPin size={18} /></span>
                        <span className="suggestion-copy"><strong>{suggestion.name}</strong><small>{suggestion.address}</small></span>
                        <span className="suggestion-add">Add</span>
                      </button>
                    </li>
                  ))}
                </ul>
                {visibleResult && (
                  <p className="place-attribution">
                    <a href={visibleResult.attribution.provider_url} target="_blank" rel="noreferrer">{visibleResult.attribution.provider}</a>
                    <span aria-hidden="true"> · </span>
                    <a href={visibleResult.attribution.data_url} target="_blank" rel="noreferrer">{visibleResult.attribution.data}</a>
                  </p>
                )}
              </div>
            )}
          </div>
        </form>
      </section>

      <section className="saved-place-library" aria-labelledby="saved-places-title">
        <div className="section-heading saved-places-heading">
          <div><p className="eyebrow">Your collection</p><h2 id="saved-places-title">Saved places</h2></div>
          <label className="search-box saved-place-filter">
            <Search size={18} />
            <input value={savedQuery} onChange={(event) => setSavedQuery(event.target.value)} placeholder="Filter saved places" />
          </label>
        </div>

        {filtered.length ? (
          <section className="place-grid">
            {filtered.map((place) => (
              <article className="place-card" key={place.localId}>
                <div className="place-card-pin"><MapPin size={20} /></div>
                <div>
                  <div className="place-card-heading"><h2>{place.name}</h2><SyncBadge state={place.syncState} /></div>
                  {place.notes && <p>{place.notes}</p>}
                  <small>{place.latitude.toFixed(5)}, {place.longitude.toFixed(5)}</small>
                </div>
                <a href={`https://www.openstreetmap.org/?mlat=${place.latitude}&mlon=${place.longitude}#map=16/${place.latitude}/${place.longitude}`} target="_blank" rel="noreferrer" aria-label={`Open ${place.name} on map`}><ExternalLink size={17} /></a>
              </article>
            ))}
          </section>
        ) : (
          <EmptyState title={savedQuery ? "No matching saved places" : "Build your personal map"} copy={savedQuery ? "Try another saved name or note." : "Search ahead or pin a café, viewpoint, hotel, or any place you want to find again."} />
        )}
        <p className="place-attribution library-attribution">
          Place and map data may include <a href="https://www.geoapify.com/" target="_blank" rel="noreferrer">Geoapify</a>
          <span aria-hidden="true"> · </span>
          <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noreferrer">© OpenStreetMap contributors</a>
        </p>
      </section>

      {showForm && (
        <div className="modal-backdrop" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && closeForm()}>
          <section className="modal-card" role="dialog" aria-modal="true" aria-labelledby="place-title">
            <button className="modal-close" onClick={closeForm} aria-label="Close">×</button>
            <p className="eyebrow">{draftSource === "search" ? "Advance pin" : "Standalone pin"}</p>
            <h2 id="place-title">{draftSource === "search" ? "Save this place" : "Save where you are"}</h2>
            {busy && !coordinates && <InlineMessage>Getting an accurate location…</InlineMessage>}
            {error && <InlineMessage tone="error">{error}</InlineMessage>}
            {coordinates && (
              <form onSubmit={save}>
                <PlacePreviewMap coordinates={coordinates} name={name} />
                {selectedAddress && <p className="selected-place-address"><MapPin size={16} /> <span>{selectedAddress}</span></p>}
                <p className="coordinate-preview">
                  <MapPin size={16} /> {coordinates.latitude.toFixed(5)}, {coordinates.longitude.toFixed(5)}
                  {coordinates.accuracy !== undefined && <> · ±{Math.round(coordinates.accuracy)} m</>}
                </p>
                <label className="field"><span>Name</span><input value={name} maxLength={200} required autoFocus onChange={(event) => setName(event.target.value)} /></label>
                <label className="field"><span>Notes</span><textarea value={notes} rows={3} onChange={(event) => setNotes(event.target.value)} placeholder="Why should you come back?" /></label>
                <button className="button primary wide" disabled={busy || !name.trim()}><MapPin size={18} /> {busy ? "Saving…" : "Save place"}</button>
              </form>
            )}
          </section>
        </div>
      )}
    </div>
  );
}
