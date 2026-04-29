# Phase 1 - Dataset, business capabilities and use cases

## Dataset

- **Primary dataset**: [Large Car Dataset on Kaggle](https://www.kaggle.com/datasets/cisautomotiveapi/large-car-dataset)
- **Domain**: Used and new car marketplace listings
- **Approximate size**: 5.3 GB
- **Release window**: Published in 2020 and updated in 2021
- **Project usage**: The dataset is prepared locally through the scripts under `scripts/local/` and loaded into the listing-related services and databases used by the platform.

## Business capabilities

- **Listing catalog and details**: Expose detailed vehicle listings and allow side-by-side comparison of multiple cars.
- **Filtered search**: Search listings using the implemented filters available in the gateway, namely make, model, year, price range, mileage, fuel type, pagination, and sold-state inclusion.
- **Market price analysis**: Calculate average market price information for a given brand and model, with optional year range filters.
- **Geographical market insights**: Compare prices by district, city, or country and retrieve aggregate metrics or location-specific statistics.
- **Seller management and profiling**: Retrieve seller profiles and distinguish marketplace actors at seller level.
- **User registration and favorites**: Register users, authenticate them with JWT, and allow them to manage favorite listings.
- **Buyer-seller communication**: Open chat sessions tied to listings and exchange real-time messages over WebSockets.
- **Auction module**: Create auctions for listings, place bids, retrieve bids, and receive real-time auction updates.
- **Listing lifecycle management**: Support authenticated listing creation, update, and deletion through the gateway in addition to read-only marketplace operations.

## Use cases

- **Use Case 1 - Market Price Analysis**: A visitor wants to know the average market price for a specific vehicle model. The platform returns aggregated price information for the selected brand and model, optionally constrained by a year interval.
- **Use Case 2 - Filtered Search**: A visitor searches the catalog using filters such as make, model, year, price range, mileage, and fuel type, and receives paginated matching listings.
- **Use Case 3 - Geographical Market Insights**: A visitor compares how the same vehicle model behaves across districts, cities, or countries, using aggregate metrics and location-specific statistics.
- **Use Case 4 - Seller Management and Profiling**: A visitor opens a listing and wants to inspect the seller profile associated with it.
- **Use Case 5 - Buyer-Seller Communication**: An authenticated user opens a chat for a specific listing and exchanges messages with the seller in real time.
- **Use Case 6 - Visitor and User Registration**: A visitor registers, logs in, and saves or removes favorite listings from their profile.
- **Use Case 7 - Auction Module**: An authenticated seller creates an auction for a listing, while authenticated buyers place bids and subscribed clients receive live auction updates.
- **Use Case 8 - Listing Details and Comparison**: A visitor views a single listing in detail or compares several selected listings side by side.

## Contribution plan

- **Francisco Encarnação**
  - Use Case 1 - Market Price Analysis
  - Use Case 7 - Auction Module
- **Daniel Carvalho**
  - Use Case 2 - Filtered Search
  - Use Case 8 - Listing Details and Comparison
- **Daniel Nunes**
  - Use Case 6 - Visitor and User Registration
  - Use Case 4 - Seller Management and Profiling
- **Daniel Sousa**
  - Use Case 5 - Buyer-Seller Communication
  - Use Case 3 - Geographical Market Insights

## Possible extensions

- **Visual search**: Allow users to upload a vehicle photo and retrieve similar listings by integrating a computer vision API.
- **Image enrichment**: Attach external labeled car image datasets to listings so each vehicle record can be complemented with reference imagery.
- **Cloud-native observability**: Extend the current Kubernetes deployment with centralized metrics, tracing, and alerting for gateway and service-level behavior.
