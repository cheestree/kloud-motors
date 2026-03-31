# computacao-nuvem-2025

To run this branch, you need to have Docker installed on your machine. Follow the instructions below to set up and run the project:

1. Clone the repository:

   ```bash
   git clone
    ```

2. Navigate to the project directory:

   ```bash
   cd computacao-nuvem-2025
   ```

3. Download the dataset and place it in the `setup` directory. You can download the dataset from [this link](https://www.kaggle.com/datasets/cisautomotiveapi/large-car-dataset).

4. Run the following command to prepare the dataset:

    ```bash
    ./prepare.sh
    ```

5. When the dataset is prepared, you can start the services with the following command:

    ```bash
    ./start.sh
    ```

6. Subsequently, the database will be populated with:

    ```bash
    ./seed.sh
    ```

The services will be available at the following gRPC endpoints:

- Searching: `localhost:50051`
- Listing: `localhost:50052`
