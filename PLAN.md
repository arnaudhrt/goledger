# GoLedger — Implementation Plan

## Step 1: Foundation (DB + Config + Project structure)

- Create the `internal/` package structure (`db/`, `config/`, `currency/`, `parser/`, `app/`)
- Implement SQLite storage: init DB, create tables (`entries`, `rates`), CRUD for entries
- Implement TOML config loader: read/write `~/.spend/config.toml`, default categories
- Wire up: app startup creates `~/.spend/` dir, opens DB, loads config

**Done when:** `go run .` boots, creates `~/.spend/spend.db` and `config.toml` with defaults, then exits cleanly.

---

## Step 2: Monthly Dashboard (Screen 1)

- Build the main Bubbletea app model with screen routing
- Implement dashboard layout: header with month nav, three-lane bar (income/expense/investment), category breakdown with horizontal bars, recent entries list
- Month navigation with `<-` / `->`
- Toggle subcategory expansion with `t`
- Scroll support for categories and entries

**Done when:** Dashboard renders with real data from DB (even if empty), month nav works, layout matches spec.

---

## Step 3: Bulk Paste (Screens 2 + 3)

- Build the line parser (`internal/parser/`): parse `DD/MM TYPE description amount [currency] [category]`
- Handle formatting quirks: commas in numbers, optional currency, optional category
- Build bulk paste screen: text area input, live preview with status indicators
- Build category assignment sub-screen: fuzzy picker for uncategorized entries
- Save confirmed entries to DB

**Done when:** Can paste a block of entries from the spec examples, assign categories, and see them saved in the DB.

---

## Step 4: Multi-currency

- Implement frankfurter.app API client (`internal/currency/`)
- Auto-fetch rates on startup, cache in `rates` table for 24h
- Convert amounts to display currency for dashboard totals and bars
- Show current rates in dashboard header
- Offline fallback to last cached rates

**Done when:** Entries in USD/EUR/HKD display correctly converted to THB on the dashboard.

---

## Step 5: Add Single Entry (Screen 5) + Category Drilldown (Screen 4)

- Build add-entry form overlay: type toggle, amount input, currency cycle, fuzzy category picker, date picker, note field
- Build category drilldown screen: subcategory bars, entry list, footer stats (avg/entry, count, avg/day)
- Entry delete (`d`) and edit (`e`) from drilldown

**Done when:** Can add entries via form, drill into any category, edit/delete entries.

---

## Step 6: Settings (Screen 6) + Polish

- Build settings screen: display currency selector, auto-fetch toggle, rate table, manual refresh (`r`), category list
- Polish: color coding (green/red/blue), box-drawing, spacing, responsive terminal sizing
- Edge cases: empty months, zero amounts, long notes truncation
- Final keybinding audit against spec

**Done when:** All 6 screens functional, all keybindings from spec work, app feels complete.

---

## Current status

- [x] Project scaffold (go.mod, basic bubbletea main.go)
- [x] **Step 1** — Foundation
- [x] **Step 2** — Dashboard
- [x] **Step 3** — Bulk Paste
- [ ] Step 4 — Multi-currency
- [ ] Step 5 — Add Entry + Drilldown
- [ ] Step 6 — Settings + Polish
