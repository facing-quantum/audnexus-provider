# -*- coding: utf-8 -*-
# Audnexus Agent Shim for Plex Media Server
# Proxies search and metadata calls to the modern audnexus-provider Go service.

import json
import os

DEFAULT_PROVIDER_URL = 'http://localhost:8080'


def get_provider_url():
    """Retrieve the configured audnexus-provider base URL.
    Checks AUDNEXUS_PROVIDER_URL and AUDNEXUS_URL environment variables first,
    then Plex agent preferences (Prefs['provider_url']),
    falling back to DEFAULT_PROVIDER_URL (http://localhost:8080).
    """
    # 1. Environment variables
    for env_var in ('AUDNEXUS_PROVIDER_URL', 'AUDNEXUS_URL'):
        val = os.environ.get(env_var)
        if val and val.strip():
            return val.strip().rstrip('/')

    # 2. Plex Agent Preferences (configured via Plex Web)
    try:
        url = Prefs['provider_url']
        if url and url.strip():
            return url.strip().rstrip('/')
    except Exception:
        pass

    # 3. Default fallback
    return DEFAULT_PROVIDER_URL


def http_get_json(url):
    """Make an HTTP GET request and return parsed JSON."""
    try:
        response = HTTP.Request(url, timeout=15)
        content = response.content
        if hasattr(content, 'decode'):
            content = content.decode('utf-8')
        return json.loads(content)
    except Exception as e:
        Log.Error('Audnexus shim GET error for %s: %s' % (url, str(e)))
        return None


def http_post_json(url, data_dict):
    """Make an HTTP POST request with JSON payload and return parsed JSON."""
    try:
        payload = json.dumps(data_dict)
        headers = {
            'Content-Type': 'application/json',
            'Accept': 'application/json'
        }
        response = HTTP.Request(url, data=payload, headers=headers, timeout=20)
        content = response.content
        if hasattr(content, 'decode'):
            content = content.decode('utf-8')
        return json.loads(content)
    except Exception as e:
        Log.Error('Audnexus shim POST error for %s: %s' % (url, str(e)))
        return None


def Start():
    Log.Info('Starting Audnexus Agent Shim v2.0.0')
    HTTP.Headers['User-Agent'] = 'Audnexus-Shim/2.0.0'


def ValidatePrefs():
    Log.Info('Audnexus shim preferences updated: provider_url=%s' % get_provider_url())


class AudiobookArtist(Agent.Artist):
    name = 'Audnexus'
    languages = [Locale.Language.English, 'de', 'es', 'fr', 'it', 'ja']
    primary_provider = True
    accepts_from = ['com.plexapp.agents.localmedia']

    def search(self, results, media, lang, manual):
        artist_name = media.artist or media.title
        if not artist_name:
            Log.Debug('No artist name provided for search')
            return

        Log.Info('Searching artist: "%s" (manual=%s)' % (artist_name, str(manual)))
        base_url = get_provider_url()
        matches_url = '%s/audnexus/library/metadata/matches' % base_url

        req_body = {
            'type': 'artist',
            'title': artist_name,
            'manual': 1 if manual else 0
        }

        data = http_post_json(matches_url, req_body)
        if not data:
            return

        container = data.get('MediaContainer', {})
        items = container.get('Metadata', [])

        for item in items:
            key = item.get('ratingKey') or item.get('key')
            name = item.get('title', artist_name)
            score = int(item.get('score', 80))
            if key:
                results.Append(MetadataSearchResult(
                    id=str(key),
                    name=name,
                    score=score,
                    lang=lang
                ))

    def update(self, metadata, media, lang, force):
        Log.Info('Updating artist: id=%s title=%s' % (metadata.id, media.title))
        base_url = get_provider_url()
        item_url = '%s/audnexus/library/metadata/%s' % (base_url, metadata.id)

        data = http_get_json(item_url)
        if not data:
            return

        container = data.get('MediaContainer', {})
        items = container.get('Metadata', [])
        if not items:
            return

        item = items[0]

        if item.get('title'):
            metadata.title = item['title']
        if item.get('summary'):
            metadata.summary = item['summary']

        # Genres
        if 'Genre' in item and isinstance(item['Genre'], list):
            metadata.genres.clear()
            for g in item['Genre']:
                tag = g.get('tag') if isinstance(g, dict) else g
                if tag:
                    metadata.genres.add(tag)

        # Poster / Thumbnail artwork
        thumb_url = item.get('thumb')
        if thumb_url and (thumb_url not in metadata.posters or force):
            try:
                metadata.posters[thumb_url] = Proxy.Media(
                    HTTP.Request(thumb_url).content, sort_order=0
                )
            except Exception as e:
                Log.Warn('Failed to fetch artist thumb: %s' % str(e))


class AudiobookAlbum(Agent.Album):
    name = 'Audnexus'
    languages = [Locale.Language.English, 'de', 'es', 'fr', 'it', 'ja']
    primary_provider = True
    accepts_from = ['com.plexapp.agents.localmedia']

    def search(self, results, media, lang, manual):
        album_title = media.album or media.title
        author_name = ''
        if hasattr(media, 'parent_metadata') and media.parent_metadata and media.parent_metadata.title:
            author_name = media.parent_metadata.title
        elif hasattr(media, 'artist') and media.artist:
            author_name = media.artist

        if not album_title:
            Log.Debug('No album title provided for search')
            return

        Log.Info('Searching album: "%s" by "%s" (manual=%s)' % (album_title, author_name, str(manual)))
        base_url = get_provider_url()
        matches_url = '%s/audnexus/library/metadata/matches' % base_url

        req_body = {
            'type': 'album',
            'title': album_title,
            'author': author_name,
            'manual': 1 if manual else 0
        }

        data = http_post_json(matches_url, req_body)
        if not data:
            return

        container = data.get('MediaContainer', {})
        items = container.get('Metadata', [])

        for item in items:
            key = item.get('ratingKey') or item.get('key')
            title = item.get('title', album_title)
            score = int(item.get('score', 80))
            year = item.get('year')

            # Build readable match label: Title by Author
            author = item.get('parentTitle') or author_name
            name = ('"%s" by %s' % (title, author)) if author else title

            if key:
                res_kwargs = {
                    'id': str(key),
                    'name': name,
                    'score': score,
                    'lang': lang
                }
                if year:
                    try:
                        res_kwargs['year'] = int(year)
                    except (ValueError, TypeError):
                        pass
                results.Append(MetadataSearchResult(**res_kwargs))

    def update(self, metadata, media, lang, force):
        Log.Info('Updating album: id=%s title=%s' % (metadata.id, media.title))
        base_url = get_provider_url()
        item_url = '%s/audnexus/library/metadata/%s' % (base_url, metadata.id)

        data = http_get_json(item_url)
        if not data:
            return

        container = data.get('MediaContainer', {})
        items = container.get('Metadata', [])
        if not items:
            return

        item = items[0]

        if item.get('title'):
            metadata.title = item['title']
        if item.get('titleSort'):
            metadata.title_sort = item['titleSort']
        if item.get('studio'):
            metadata.studio = item['studio']
        if item.get('summary'):
            metadata.summary = item['summary']

        # Rating (convert out of 10 or 5 to Plex float)
        if item.get('rating') is not None:
            try:
                metadata.rating = float(item['rating'])
            except (ValueError, TypeError):
                pass

        # Release date
        if item.get('originallyAvailableAt'):
            try:
                metadata.originally_available_at = Datetime.ParseDate(item['originallyAvailableAt']).date()
                metadata.year = metadata.originally_available_at.year
            except Exception:
                if item.get('year'):
                    try:
                        metadata.year = int(item['year'])
                    except (ValueError, TypeError):
                        pass

        # Genres (mapped from item.Genre)
        if 'Genre' in item and isinstance(item['Genre'], list):
            metadata.genres.clear()
            for g in item['Genre']:
                tag = g.get('tag') if isinstance(g, dict) else g
                if tag:
                    metadata.genres.add(tag)

        # Moods (mapped from item.Mood)
        if 'Mood' in item and isinstance(item['Mood'], list):
            metadata.moods.clear()
            for m in item['Mood']:
                tag = m.get('tag') if isinstance(m, dict) else m
                if tag:
                    metadata.moods.add(tag)

        # Poster Artwork
        thumb_url = item.get('thumb')
        if thumb_url and (thumb_url not in metadata.posters or force):
            try:
                metadata.posters[thumb_url] = Proxy.Media(
                    HTTP.Request(thumb_url).content, sort_order=0
                )
                metadata.posters.validate_keys([thumb_url])
            except Exception as e:
                Log.Warn('Failed to fetch album cover: %s' % str(e))
