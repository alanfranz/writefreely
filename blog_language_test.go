/*
 * Copyright © 2026 Musing Studio LLC.
 *
 * This file is part of WriteFreely.
 *
 * WriteFreely is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, included
 * in the LICENSE file in this source code package.
 */

package writefreely

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/writeas/impart"
	"github.com/writeas/web-core/converter"

	"github.com/writefreely/writefreely/config"
)

func TestNormalizeLanguage(t *testing.T) {
	for in, want := range map[string]string{
		"":     "",
		"  ":   "",
		"it":   "it",
		" IT ": "it",
		"he":   "he",
	} {
		got, ok := normalizeLanguage(in)
		assert.True(t, ok, "%q should be valid", in)
		assert.Equal(t, want, got, "normalizing %q", in)
	}
	for _, in := range []string{"xx", "ita", "en-US", "en_GB", "i", "<b>"} {
		_, ok := normalizeLanguage(in)
		assert.False(t, ok, "%q should be invalid", in)
	}
}

func TestPostLanguagesList(t *testing.T) {
	codeRE := regexp.MustCompile(`^[a-z]{2}$`)
	seen := map[string]bool{}
	for _, l := range postLanguages {
		assert.Regexp(t, codeRE, l.Code, "codes must fit posts.language CHAR(2)")
		assert.NotEmpty(t, l.Name, "language %s has no name", l.Code)
		assert.False(t, seen[l.Code], "duplicate language %s", l.Code)
		seen[l.Code] = true
	}
	assert.True(t, seen["it"] && seen["en"])
}

// blogLangTestEnv is a multi-user instance with one user and their blog, which
// has one post that was created before the blog had a language.
type blogLangTestEnv struct {
	app    *App
	router *mux.Router
	user   *User
	coll   *Collection
	// oldPost was created before a blog language was set, without a language.
	oldPost *Post
}

func newBlogLangTestEnv(t *testing.T) *blogLangTestEnv {
	t.Helper()
	app, router := newTemplateTestApp(t, func(cfg *config.Config) {
		cfg.App.SingleUser = false
		cfg.App.Chorus = false
	})
	u, coll, post := createTemplateTestUser(t, app, "linguist")
	require.False(t, post.Language.Valid, "precondition: post shouldn't have a language")
	return &blogLangTestEnv{app: app, router: router, user: u, coll: coll, oldPost: post}
}

func (e *blogLangTestEnv) setBlogLanguage(t *testing.T, lang string, applyToPosts bool) {
	t.Helper()
	err := e.app.db.UpdateCollection(e.app, &SubmittedCollection{
		OwnerID:        uint64(e.user.ID),
		Language:       &lang,
		LangApplyPosts: applyToPosts,
	}, e.coll.Alias)
	require.NoError(t, err)
}

func (e *blogLangTestEnv) blogLanguage(t *testing.T) string {
	t.Helper()
	c, err := e.app.db.GetCollection(e.coll.Alias)
	require.NoError(t, err)
	return c.Language
}

func (e *blogLangTestEnv) storedPostLanguage(t *testing.T, id string) sql.NullString {
	t.Helper()
	var lang sql.NullString
	require.NoError(t, e.app.db.QueryRow("SELECT language FROM posts WHERE id = ?", id).Scan(&lang))
	return lang
}

// createPost creates a post on the blog the way the web editor does, i.e.
// submitting the browser's UI language.
func (e *blogLangTestEnv) createPost(t *testing.T, collID int64, browserLang, title, content string) *Post {
	t.Helper()
	p, err := e.app.db.CreatePost(e.user.ID, collID, &SubmittedPost{
		Title:    &title,
		Content:  &content,
		Language: converter.NullJSONString{NullString: sql.NullString{String: browserLang, Valid: browserLang != ""}},
	})
	require.NoError(t, err)
	return p
}

func (e *blogLangTestEnv) get(t *testing.T, path string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "GET %s: %s", path, truncateForLog(rec.Body.String()))
	return rec
}

func TestBlogLanguage_Setting(t *testing.T) {
	e := newBlogLangTestEnv(t)
	assert.Equal(t, "", e.blogLanguage(t), "blogs have no language by default")

	e.setBlogLanguage(t, " IT ", false)
	assert.Equal(t, "it", e.blogLanguage(t), "language should be normalized")
	assert.False(t, e.storedPostLanguage(t, e.oldPost.ID).Valid, "existing posts are left alone unless asked")

	// Unsupported values are rejected, leaving the current value in place
	bad := "xx"
	err := e.app.db.UpdateCollection(e.app, &SubmittedCollection{OwnerID: uint64(e.user.ID), Language: &bad}, e.coll.Alias)
	if assert.Error(t, err) {
		herr, ok := err.(impart.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, herr.Status)
	}
	assert.Equal(t, "it", e.blogLanguage(t))

	// Updates that don't include a language leave it alone
	title := "Retitled"
	require.NoError(t, e.app.db.UpdateCollection(e.app, &SubmittedCollection{OwnerID: uint64(e.user.ID), Title: &title}, e.coll.Alias))
	assert.Equal(t, "it", e.blogLanguage(t))

	// An empty value unsets it
	e.setBlogLanguage(t, "", true)
	assert.Equal(t, "", e.blogLanguage(t))
	assert.False(t, e.storedPostLanguage(t, e.oldPost.ID).Valid, "unsetting the language shouldn't touch posts")
}

func TestBlogLanguage_ApplyToExistingPosts(t *testing.T) {
	e := newBlogLangTestEnv(t)
	frPost := e.createPost(t, e.coll.ID, "fr", "Bonjour", "Le contenu.")
	draft := e.createPost(t, 0, "fr", "Brouillon", "Pas sur le blog.")

	e.setBlogLanguage(t, "it", true)

	assert.Equal(t, "it", e.storedPostLanguage(t, e.oldPost.ID).String)
	assert.Equal(t, "it", e.storedPostLanguage(t, frPost.ID).String)
	assert.Equal(t, "fr", e.storedPostLanguage(t, draft.ID).String, "posts outside the blog must not change")
}

func TestBlogLanguage_NewAndMovedPosts(t *testing.T) {
	e := newBlogLangTestEnv(t)

	// Without a blog language, the submitted language is kept as-is
	p := e.createPost(t, e.coll.ID, "en", "Before", "Posted before setting a blog language.")
	assert.Equal(t, "en", p.Language.String)

	e.setBlogLanguage(t, "de", false)

	// The blog language takes precedence over the browser's
	p = e.createPost(t, e.coll.ID, "en", "Über uns", "Wir schreiben auf Deutsch.")
	assert.Equal(t, "de", p.Language.String)
	assert.Equal(t, "de", e.storedPostLanguage(t, p.ID).String)
	assert.Equal(t, "ueber-uns", p.Slug.String, "slug should be transliterated using the blog's language")

	p = e.createPost(t, e.coll.ID, "", "No browser language", "Posted via the API.")
	assert.Equal(t, "de", e.storedPostLanguage(t, p.ID).String)

	// Drafts aren't on a blog, so they keep the submitted language...
	draft := e.createPost(t, 0, "en", "Entwurf", "Noch nicht veröffentlicht.")
	assert.Equal(t, "en", e.storedPostLanguage(t, draft.ID).String)

	// ...until they're moved to the blog
	res, err := e.app.db.ClaimPosts(e.app.cfg, e.user.ID, e.coll.Alias, &[]ClaimPostRequest{{AnonymousAuthPost: &AnonymousAuthPost{ID: draft.ID}}})
	require.NoError(t, err)
	require.Len(t, *res, 1)
	require.Equal(t, http.StatusOK, (*res)[0].Code, (*res)[0].ErrorMessage)
	assert.Equal(t, "de", (*res)[0].Post.Language.String)
	assert.Equal(t, "de", e.storedPostLanguage(t, draft.ID).String)
}

func TestBlogLanguage_Rendering(t *testing.T) {
	e := newBlogLangTestEnv(t)
	prefix := "/" + e.coll.Alias
	tagged := e.createPost(t, e.coll.ID, "", "Tagged", "A post with a #lingua tag.")
	// Tag pages only list posts created strictly in the past
	_, err := e.app.db.Exec("UPDATE posts SET created = ? WHERE id = ?", time.Now().UTC().Add(-time.Hour), tagged.ID)
	require.NoError(t, err)

	// Without a blog language, nothing is announced
	rec := e.get(t, prefix+"/", nil)
	assert.Empty(t, rec.Header().Get("Content-Language"))
	assert.NotContains(t, rec.Body.String(), `<html lang=`)
	rec = e.get(t, prefix+"/feed/", nil)
	assert.NotContains(t, rec.Body.String(), "<language>")

	e.setBlogLanguage(t, "it", false)
	// A post with its own language, e.g. set from its "Edit metadata" page
	frPost := e.createPost(t, e.coll.ID, "", "Bonjour", "Ce billet est en français.")
	_, err = e.app.db.Exec("UPDATE posts SET language = 'fr' WHERE id = ?", frPost.ID)
	require.NoError(t, err)

	// Blog pages are in the blog's language
	for _, path := range []string{prefix + "/", prefix + "/archive/", prefix + "/tag:lingua"} {
		rec := e.get(t, path, nil)
		assert.Equal(t, "it", rec.Header().Get("Content-Language"), path)
		assert.Contains(t, rec.Body.String(), `<html lang="it" dir="ltr">`, path)
	}

	// The /lang:{lang} pages are still in the requested language
	rec = e.get(t, prefix+"/lang:fr", nil)
	assert.Equal(t, "fr", rec.Header().Get("Content-Language"))
	assert.Contains(t, rec.Body.String(), `<html lang="fr"`)

	// A post without a language of its own is rendered in the blog's language
	for _, slug := range []string{e.oldPost.Slug.String, tagged.Slug.String} {
		rec = e.get(t, prefix+"/"+slug, nil)
		assert.Equal(t, "it", rec.Header().Get("Content-Language"), slug)
		assert.Contains(t, rec.Body.String(), `<html lang="it"`, slug)
		rec = e.get(t, prefix+"/"+slug+".md", nil)
		assert.Equal(t, "it", rec.Header().Get("Content-Language"), slug)
	}

	// ...and a post with its own language keeps it
	rec = e.get(t, prefix+"/"+frPost.Slug.String, nil)
	assert.Equal(t, "fr", rec.Header().Get("Content-Language"))
	assert.Contains(t, rec.Body.String(), `<html lang="fr"`)

	// RSS feed
	rec = e.get(t, prefix+"/feed/", nil)
	assert.Equal(t, "it", rec.Header().Get("Content-Language"))
	assert.Contains(t, rec.Body.String(), "<language>it</language>")
	assert.Contains(t, rec.Body.String(), `<rss version="2.0"`)

	// ActivityPub objects
	apHeader := http.Header{"Accept": {"application/activity+json"}}
	for slug, want := range map[string]string{e.oldPost.Slug.String: "it", frPost.Slug.String: "fr"} {
		rec = e.get(t, prefix+"/"+slug, apHeader)
		var obj struct {
			ContentMap map[string]string `json:"contentMap"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &obj), rec.Body.String())
		assert.Len(t, obj.ContentMap, 1, slug)
		assert.Contains(t, obj.ContentMap, want, slug)
	}

	// The language is exposed via the collections API
	rec = e.get(t, "/api/collections/"+e.coll.Alias, http.Header{"Content-Type": {"application/json"}})
	assert.Contains(t, rec.Body.String(), `"lang":"it"`)
}

func TestBlogLanguage_SettingsPage(t *testing.T) {
	e := newBlogLangTestEnv(t)
	cookie := loginCookie(t, e.app, e.user)

	rec := assertRendersCleanly(t, e.router, "GET", "/me/c/"+e.coll.Alias, []*http.Cookie{cookie}, http.StatusOK)
	assert.Contains(t, rec.Body.String(), `<select name="lang"`)
	assert.Contains(t, rec.Body.String(), `<option value="">(not set)</option>`)
	assert.NotContains(t, rec.Body.String(), `selected="selected">Italiano</option>`)

	// Submit the settings form, like a browser would
	form := url.Values{"lang": {"it"}, "lang_apply_posts": {"on"}, "title": {"Blog"}}
	req := httptest.NewRequest("POST", "/api/collections/"+e.coll.Alias, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusFound, w.Code, w.Body.String())
	assert.Equal(t, "it", e.blogLanguage(t))
	assert.Equal(t, "it", e.storedPostLanguage(t, e.oldPost.ID).String)

	rec = assertRendersCleanly(t, e.router, "GET", "/me/c/"+e.coll.Alias, []*http.Cookie{cookie}, http.StatusOK)
	assert.Contains(t, rec.Body.String(), `<option value="it" selected="selected">Italiano</option>`)
	assert.Contains(t, rec.Body.String(), `<option dir="rtl" value="he">`)

	// Invalid values are reported back to the user
	form = url.Values{"lang": {"xx"}}
	req = httptest.NewRequest("POST", "/api/collections/"+e.coll.Alias, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "it", e.blogLanguage(t))
	// The error is flashed in the session
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName {
			cookie = c
		}
	}
	rec = assertRendersCleanly(t, e.router, "GET", "/me/c/"+e.coll.Alias, []*http.Cookie{cookie}, http.StatusOK)
	assert.Contains(t, rec.Body.String(), "Unsupported blog language")
}

func TestBlogLanguage_API(t *testing.T) {
	e := newBlogLangTestEnv(t)
	token, err := e.app.db.GetAccessToken(e.user.ID)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/collections/"+e.coll.Alias, strings.NewReader(`{"lang": "pt"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "pt", e.blogLanguage(t))

	// Posts created via the API get the blog language
	req = httptest.NewRequest("POST", "/api/collections/"+e.coll.Alias+"/posts", strings.NewReader(`{"title": "Olá", "body": "Mundo", "lang": "en"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w = httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var res struct {
		Data struct {
			Language string `json:"language"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	assert.Equal(t, "pt", res.Data.Language)

	req = httptest.NewRequest("POST", "/api/collections/"+e.coll.Alias, strings.NewReader(`{"lang": "klingon"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w = httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "pt", e.blogLanguage(t))
}
