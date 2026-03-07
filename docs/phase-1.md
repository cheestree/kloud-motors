# Phase 1 - Dataset, business capabilities and use cases

## Dataset

- **URL**: <https://www.kaggle.com/datasets/cisautomotiveapi/large-car-dataset>
- **Topic**: Car listings. Detailed information on used and new cars for sale
- **Size**: 5.3 GB
- **Release date**: 2020 (Last updated in 2021)

## Business capabilities

- **Market Price Analysis**: Calculate the average price of cars based on brand, model, year, and location.
- **Filtered Search**: Allow users to search for cars based on specific criteria(make, model, year, price range, mileage, fuel type, location).
- **Geographical Market Insights**: Analyze car availability and price variations across different regions.
- **Seller Management & Profiling**: Distinguish between professional dealers and private sellers.
- **Buyer–Seller Communication**: Enable real-time messaging between the buyer and the seller of a specific listing.
- **Visitor & User Registration**: Track and register platform visitors, registered users can save favourite listings.
- **Auction Module**: Allow sellers to list cars under an auction format.
- **Listing details and comparison**: Provide detailed information on each listing and allow users to compare multiple listings side by side based on price, mileage, year and location.

## Use Cases

- **Use Case 1 - Market Price Analysis**: A user wants to know the average price of a car in New York. The system will analyze the dataset to provide an average price based on similar listings in that location.
- **Use Case 2 - Filtered Search**: A user is looking for a used car with less than 50,000 miles and a price range of 10,000 to 15,000$. The system will filter the dataset to show relevant listings that match these criteria.
- **Use Case 3 - Geographical Market Insights**: A user wants to compare the average price of a specific car model in different cities. The system will analyze the dataset to provide insights into price variations across locations.
- **Use Case 4 - Seller Management & Profiling**: A user wants to know if a listing is from a professional dealer or a private seller. The system will analyze the dataset to categorize listings accordingly.
- **Use Case 5 - Buyer–Seller Communication**: A user is interested in a specific car listing and wants to ask the seller a question. The system will enable real-time messaging between the buyer and the seller.
- **Use Case 6 - Visitor & User Registration**: A user visits the platform and wants to save a listing for later. The system will prompt the user to register or log in to save the listing to their profile.
- **Use Case 7 - Auction Module**: A seller wants to list a car for auction. The system will allow the seller to create an auction listing, and buyers can place bids on the car until the auction ends.
- **Use Case 8 - Listing details and comparison**: A user wants to compare multiple car listings side by side based on price, mileage, year and location. The system will retrieve and present a comparison of the selected listings.

## Contribution Plan

- Francisco Encarnação:
  - Use Case 1 – Market Price Analysis
  - Use Case 7 – Auction Module
- Daniel Carvalho:
  - Use Case 2 – Filtered Search
  - Use Case 8 – Listing Comparison
- Daniel Nunes:
  - Use Case 6 – Visitor & User Registration
  - Use Case 4 – Seller Management & Profiling
- Daniel Sousa:
  - Use Case 5 – Buyer–Seller Communication
  - Use Case 3 – Geographical Market Insights

## Possible Extensions

The following extensions may be incorporated into the project if approved.

- **Visual Search via Vision API**: Integration of a Vision API (e.g. Google Vision or a similar service) to allow users to upload a photo of a car and automatically find equal or similar models in the dataset.
- **Car Image Dataset**: Integration of a secondary dataset of labelled car images to enrich listings with reference images grouped by make, model, and year.
