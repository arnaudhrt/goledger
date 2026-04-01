# GoLedger — TUI expense tracker

A minimal Bubbletea TUI app to track where your money goes each month. Not an accounting tool — a monthly money radar.

## Philosophy

- **Glanceable**: open it, see your month at a glance — income vs expenses vs investments, category breakdown
- **Fast input**: primary workflow is bulk-pasting entries from Apple Notes, not form-filling
- **No overhead**: no accounts, no transfers, no reconciliation, no budgets, no balances
- **Three things only**: expenses, income, investments

## Tech stack

- **Language**: Go
- **TUI framework**: Bubbletea (charmbracelet)
- **Storage**: SQLite (`~/.spend/spend.db`) via `modernc.org/sqlite` (pure Go, no CGO)
- **Config**: TOML (`~/.spend/config.toml`)
- **Currency rates**: frankfurter.app (free, no API key, ECB data, supports THB/USD/EUR/HKD)

## Data model

### SQLite schema

```sql
CREATE TABLE entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL,            -- ISO format YYYY-MM-DD
    type TEXT NOT NULL,            -- 'EXP', 'INC', 'INV'
    category TEXT NOT NULL DEFAULT '',  -- hierarchical, e.g. 'food:coffee'
    note TEXT NOT NULL DEFAULT '',
    amount REAL NOT NULL,
    currency TEXT NOT NULL         -- original currency code (THB, USD, EUR, HKD)
);

CREATE INDEX idx_entries_date ON entries(date);
CREATE INDEX idx_entries_type ON entries(type);
CREATE INDEX idx_entries_category ON entries(category);

CREATE TABLE rates (
    base TEXT NOT NULL,
    target TEXT NOT NULL,
    rate REAL NOT NULL,
    fetched_at TEXT NOT NULL,      -- ISO datetime
    PRIMARY KEY (base, target)
);
```

### Config (TOML)

```toml
display_currency = "THB"
auto_fetch_rates = true

# Categories are auto-created on first use.
# This file is for manual bulk edits only.
# Categories listed here seed the fuzzy picker.
[categories]
exp = [
    "housing", "housing:rent", "housing:electric", "housing:water",
    "food", "food:dining", "food:grocery", "food:coffee",
    "transport", "transport:grab", "transport:fuel",
    "fun", "fun:sub", "fun:social",
    "health", "health:gym", "health:medical",
    "bills", "bills:internet", "bills:mobile",
]
inc = ["salary", "freelance", "other"]
inv = ["crypto", "stocks", "savings"]
```

## Categories

- Two levels max: `parent:child` (e.g. `food:coffee`, `bills:internet`)
- Auto-created on first use — type a new one and it exists
- Fuzzy searchable in the picker
- Stored in config.toml for seeding the picker, but the DB is the source of truth for what categories exist

## Multi-currency

- All entries store their original currency and amount
- Display currency (default THB) used for dashboard totals and bars
- Conversion uses cached rates from frankfurter.app
- Rates auto-fetched on app start, cached 24h in `rates` table
- Manual refresh with `r` in settings
- Fallback to last cached rate if offline
- API: `GET https://api.frankfurter.app/latest?from=USD&to=THB,EUR,HKD`

## Screens

### Screen 1 — Monthly dashboard (home)

The only real screen. Everything else is an overlay.

**Layout (top to bottom):**

1. **Header**: app name + month navigator (`◂ April 2026 ▸`) + current rates summary
2. **Three-lane bar**: income (green) / expenses (red) / investments (blue) as proportional segments, with "saved" amount and percentage on the right
3. **Category breakdown**: top-level expense categories with horizontal bars and percentages. Subcategories togglable with `t` — they expand inline beneath their parent showing amounts (no bars for subs)
4. **Recent entries**: last ~5 entries showing date, type badge (INC/EXP/INV), category, note, and amount in original currency

**Keybindings:**

- `a` — add single entry (form overlay)
- `b` — bulk paste (primary input)
- `←` `→` — navigate months
- `↑` `↓` — scroll entries / categories
- `enter` — drill into highlighted category
- `t` — toggle subcategory expansion
- `s` — settings
- `q` — quit

### Screen 2 — Bulk paste (primary input method)

This is how entries get added 90% of the time. Paste a block of text from Apple Notes.

**Input format:**

```
DD/MM TYPE description amount [currency] [category]
```

- `TYPE`: `EXP`, `INC`, `INV`
- `currency`: optional, defaults to display currency
- `category`: optional, hierarchical (e.g. `food:coffee`)
- Year inferred from current month context

**Examples:**

```
01/04 INC Salary 55,000 THB
01/04 EXP Internet bill 220 THB bills:internet
01/04 EXP mobile data bill 200 THB bills:mobile
01/04 EXP 7-11 Snack 39 THB
01/04 INV monthly btc investment 10,000 THB crypto
```

**Live preview below input:**

- Each parsed line shown with a status indicator
- `✓` — fully parsed with category
- `?` — parsed but missing category (needs assignment)
- `✗` — parse error (highlighted with reason)

**Category assignment flow:**

- Press `enter` to step through all `?` entries one by one
- Each shows the entry details + a fuzzy category picker
- Type to fuzzy-filter existing categories, or type a new one to create it
- `enter` to confirm, `tab` to skip (stays uncategorized)
- After all assigned, returns to preview for final confirmation

**No auto-matching.** Categories are always explicitly provided by the user, either inline in the paste format or via the picker. This is intentional — auto-matching is clunky and unreliable.

**Keybindings:**

- `enter` — step through uncategorized entries
- `ctrl+s` — confirm all and save
- `esc` — cancel

### Screen 3 — Category assignment (sub-screen of bulk paste)

Shown for each uncategorized entry during the assignment flow.

- Displays the entry being categorized (date, type, description, amount)
- Text input with fuzzy-filtered category list below
- Typing a new category that doesn't exist creates it

**Keybindings:**

- `↑` `↓` — navigate suggestions
- `enter` — confirm selection
- `tab` — skip (leave uncategorized)
- `esc` — back to preview

### Screen 4 — Category drilldown

Shown when pressing `enter` on a category from the dashboard.

**Layout:**

1. Category name + month + total amount + percentage of expenses
2. Subcategory bars (if any) — same horizontal bar style as dashboard
3. All entries for this category in the selected month
4. Footer stats: average per entry, entry count, average per day

**Keybindings:**

- `esc` — back to dashboard
- `enter` — drill into subcategory (if on a subcategory row)
- `d` — delete highlighted entry
- `e` — edit highlighted entry

### Screen 5 — Add single entry (form overlay)

Secondary input method for quick one-offs.

**Fields:**

1. Type: toggle `● expense ○ income ○ investment` (tab to cycle)
2. Amount: number input, `c` to cycle currency
3. Category: fuzzy picker
4. Note: free text (optional)
5. Date: defaults to today, `d` to change

**Keybindings:**

- `tab` — next field
- `c` — cycle currency (THB → USD → EUR → ...)
- `enter` — save
- `esc` — cancel

### Screen 6 — Settings

**Sections:**

1. **Currency**: display currency selector, auto-fetch toggle, last fetch time, manual refresh (`r`), current rate table
2. **Categories**: list all categories grouped by type (EXP/INC/INV). Points user to `config.toml` for bulk edits.

**Keybindings:**

- `r` — refresh rates from API
- `c` — cycle display currency
- `esc` — back

## File structure (suggested)

```
spend/
├── main.go
├── go.mod
├── go.sum
├── internal/
│   ├── app/            # Bubbletea app model, update, view
│   │   ├── app.go
│   │   ├── dashboard.go
│   │   ├── bulk.go
│   │   ├── assign.go
│   │   ├── drilldown.go
│   │   ├── addentry.go
│   │   └── settings.go
│   ├── db/             # SQLite operations
│   │   ├── db.go
│   │   ├── entries.go
│   │   └── rates.go
│   ├── parser/         # Bulk paste line parser
│   │   └── parser.go
│   ├── currency/       # Frankfurter API client + conversion
│   │   └── currency.go
│   └── config/         # TOML config loader
│       └── config.go
└── README.md
```

## Design principles for implementation

- **Keyboard-first**: every action reachable by keyboard, no mouse needed
- **Minimal chrome**: use box-drawing characters sparingly, let whitespace breathe
- **Color coding**: green = income, red = expenses, blue = investments, category colors for bars (coral, amber, blue, pink, purple, gray)
- **Fast startup**: fetch rates async, show cached data immediately
- **Forgiving input**: bulk parser should handle minor formatting inconsistencies (extra spaces, missing currency, comma in numbers)
