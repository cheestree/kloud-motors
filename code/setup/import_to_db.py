import os
import pandas as pd
from sqlalchemy import create_engine
import argparse

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--dataset", required=True, help="O ficheiro CSV final deduped")
    args = parser.parse_args()

    db_url = "postgresql://listing_user:listing_password@localhost:5432/listing_db"
    engine = create_engine(db_url)

    print(f"A carregar o ficheiro '{args.dataset}' para a memória...")
    df = pd.read_csv(args.dataset, low_memory=False)

    print("A importar dados para a tabela 'listings' na Base de Dados Postgres...")
    df.to_sql('listings', engine, if_exists='replace', index=False)
    
    print("Sucesso! A tabela 'listings' foi criada e populada.")

if __name__ == "__main__":
    main()
