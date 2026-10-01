# audnexus-provider

Go-based HTTP metadata provider for Plex Media Server, delivering rich audiobook and author metadata with resilient multi-provider fallback and high-resolution cover enrichment.

## Overview

`audnexus-provider` is a standalone HTTP metadata provider adhering to the official [Plex Media Provider specification](https://github.com/plexinc/tmdb-example-provider). It replaces the legacy Python 2 Framework plugin with a high-performance, resilient Go service.

### Multi-Provider Fallback Pipeline

To ensure the highest match rates and data quality, metadata resolution follows a battle-tested pipeline adapted from Audiobookshelf:

1. **Primary**: Direct Audible Catalog search paired with rich metadata from [api.audnex.us](https://api.audnex.us)
2. **Secondary**: Apple Books / iTunes Search API fallback
3. **Tertiary**: Google Books API fallback with fuzzy title/author matching
4. **Quaternary**: Open Library API fallback
5. **Safety Net**: Clean local synthetic matching (`local_<hash>`) for files without online catalog entries

### Metadata & Cover Enrichment

- **AudiobookCovers.com**: Automatically fetches high-resolution, text-clean original cover artwork when upstream covers are missing or low-resolution.
- **Series Cleaning**: Standardizes cluttered series sequence strings (e.g. `"Book 2, Dramatized Adaptation"` $\rightarrow$ `"2"`).
- **Tag Separation**: Separates genres and tags, mapping tags to Plex `Mood` attributes.

### Resiliency & Performance

- **Two-Tier Persistent Cache**: In-memory LRU cache backed by persistent disk storage (`~/.cache/audnexus`) with Stale-While-Revalidate to keep Plex responsive even during upstream outages.
- **Circuit Breakers**: Fast-fails calls to unresponsive upstreams (Audible, iTunes) with automated cooldown and health recovery.
- **Request Coalescing**: Uses `singleflight` deduplication to prevent duplicate upstream calls during bulk Plex library scans.

---

## Features

- **Plex Media Provider Compliance**: Implements the official Plex Media Provider API (`tv.plex.agents.custom.audnexus`), with top-level `MediaProvider` and `MediaContainer` JSON response envelopes.
- **All 10 Audible regions supported**: `au`, `ca`, `de`, `es`, `fr`, `in`, `it`, `jp`, `us`, `uk`.
- **ASIN quick matching**: Direct lookups by ASIN from folder/filenames for instant 100% accuracy.
- **Fuzzy search with Levenshtein scoring**: Intelligent title, author, and narrator matching.
- **Cross-platform**: Pre-built static binaries for Linux, macOS, and Windows (`amd64` + `arm64`).
- **Dedicated Image Endpoint**: Serves available poster and artwork assets via `/library/metadata/:ratingKey/images`.

---

## Installation

### Docker (GHCR)

Run directly with Docker:
```bash
docker run -d \
  --name audnexus-provider \
  -p 8080:8080 \
  -v audnexus-cache:/cache \
  -e REGION=us \
  ghcr.io/facing-quantum/audnexus.bundle:latest
```

Or using Docker Compose:
```yaml
services:
  audnexus-provider:
    image: ghcr.io/facing-quantum/audnexus.bundle:latest
    container_name: audnexus-provider
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - REGION=us
      - PORT=8080
    volumes:
      - audnexus-cache:/cache

volumes:
  audnexus-cache:
```

### Download Binary (Standalone)

Download the latest release binary for your platform from the [Releases](https://github.com/facing-quantum/Audnexus.bundle/releases) page.

### Build from Source

```bash
# Clone the repository
git clone https://github.com/facing-quantum/Audnexus.bundle.git
cd Audnexus.bundle

# Build for current platform (outputs to bin/audnexus-provider)
make build

# Or build cross-platform binaries
make build-all
```

---

## Configuration

Configuration is managed via environment variables or a `.env` file:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `REGION` | `us` | Audible region (`au`, `ca`, `de`, `es`, `fr`, `in`, `it`, `jp`, `us`, `uk`) |
| `CACHE_DIR` | `~/.cache/audnexus` | Directory for persistent disk cache |
| `CACHE_TTL` | `604800` | Cache TTL in seconds (default: 1 week) |
| `AUDNEXUS_TIMEOUT` | `90` | API request timeout in seconds |
| `KEEP_EXISTING_GENRES` | `false` | Keep existing Plex genres instead of replacing |
| `STORE_AUTHOR_AS_MOOD` | `true` | Store author names as Plex mood tags |
| `SORT_AUTHOR_BY_LAST_NAME` | `true` | Sort author names as `Last, First` |
| `SIMPLIFY_TITLE` | `false` | Simplify titles by stripping subtitles and series clutter |
| `LOG_LEVEL` | `INFO` | Log level (`DEBUG`, `INFO`, `WARN`, `ERROR`) |

Example:
```bash
export REGION=us
export PORT=8080
export LOG_LEVEL=INFO
./bin/audnexus-provider
```

---

## Usage with Plex

1. **Start the provider**:
   ```bash
   ./bin/audnexus-provider
   ```

2. **In Plex Web**, navigate to **Settings > Manage > Libraries** (or add a new Music library for audiobooks).

3. Set the metadata provider URL to:
   ```
   http://localhost:8080/audnexus
   ```

4. Refresh metadata on your audiobook library.

---

## API Endpoints

### 1. Health Check
```http
GET /health
```
Returns `{"status": "healthy"}`.

### 2. Provider Discovery
```http
GET /audnexus
```
Returns the `MediaProviderResponse` envelope with supported types (Artist: `8`, Album: `9`), GUID schemes, and feature endpoints.

### 3. Search & Matching
```http
POST /audnexus/library/metadata/matches
Content-Type: application/json

{
  "type": 9,             // 8 for Artist, 9 for Album (or "artist"/"album")
  "title": "Way of Kings",
  "author": "Brandon Sanderson",
  "manual": 1            // 1 returns multiple candidates; 0 returns best match
}
```
Returns a `MediaContainerResponse` containing matching metadata items sorted by score.

### 4. Full Metadata Retrieval
```http
GET /audnexus/library/metadata/:ratingKey
```
Example:
- `GET /audnexus/library/metadata/album_B003ZWFO7E`
- `GET /audnexus/library/metadata/author_B001IGFHW6`

Returns full metadata with structured Plex tag arrays (`Genre`, `Mood`, `Style`, `Similar`), clean series sequences, and external `Guid` cross-references.

### 5. Image Assets
```http
GET /audnexus/library/metadata/:ratingKey/images
```
Returns a `MediaContainerResponse` containing all available cover art and background images for the item.

---

## Development

```bash
# Run all unit tests
make test
# or: go test -v ./...

# Run the server locally
make run

# Clean build artifacts
make clean
```

---

## License

GPL v3.0 - See [LICENSE](LICENSE) file.