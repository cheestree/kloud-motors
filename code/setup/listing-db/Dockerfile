FROM python:3.12-slim AS data-prep

WORKDIR /data
COPY setup/common/display_columns.py ./
COPY setup/listing-db/prepare_listings.py ./
COPY setup/listing-db/dataset_original.csv ./

RUN pip install pandas --break-system-packages && \
    python3 prepare_listings.py

FROM postgres:16

RUN apt-get update && apt-get install -y python3 python3-pip && \
    pip3 install psycopg2-binary --break-system-packages

COPY --from=data-prep /data/dataset_prepared.csv /data/dataset_prepared.csv
COPY setup/listing-db/load_listings.py /scripts/load_listings.py
COPY setup/listing-db/init.sh /docker-entrypoint-initdb.d/init.sh
RUN chmod +x /docker-entrypoint-initdb.d/init.sh