CREATE INDEX IF NOT EXISTS idx_automotive_brand_model_year_price
    ON automotive_data (brand_id, model_id, model_year, ask_price);

CREATE INDEX IF NOT EXISTS idx_automotive_city
    ON automotive_data (city);

CREATE INDEX IF NOT EXISTS idx_automotive_district
    ON automotive_data (district);

CREATE INDEX IF NOT EXISTS idx_automotive_country
    ON automotive_data (country);

CREATE INDEX IF NOT EXISTS idx_automotive_state
    ON automotive_data (state);

CREATE INDEX IF NOT EXISTS idx_automotive_city_lower
    ON automotive_data (LOWER(city));

CREATE INDEX IF NOT EXISTS idx_automotive_district_lower
    ON automotive_data (LOWER(district));

CREATE INDEX IF NOT EXISTS idx_automotive_country_lower
    ON automotive_data (LOWER(country));

CREATE INDEX IF NOT EXISTS idx_automotive_state_lower
    ON automotive_data (LOWER(state));

CREATE INDEX IF NOT EXISTS idx_automotive_fuel_type_id
    ON automotive_data (fuel_type_id);

CREATE INDEX IF NOT EXISTS idx_automotive_is_sold
    ON automotive_data (is_sold);

CREATE INDEX IF NOT EXISTS idx_brand_name_lower
    ON brand (LOWER(name));

CREATE INDEX IF NOT EXISTS idx_model_name_lower
    ON model (LOWER(name));
