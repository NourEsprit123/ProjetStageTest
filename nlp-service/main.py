from fastapi import FastAPI
from pydantic import BaseModel
from sentence_transformers import SentenceTransformer

app = FastAPI()
# Chargement d'un modèle léger et performant pour le français/multilingue
model = SentenceTransformer('paraphrase-multilingual-MiniLM-L12-v2')

class TextRequest(BaseModel):
    text: str

@app.post("/vectorize")
async def vectorize(request: TextRequest):
    # Transformation du texte en vecteur
    vector = model.encode(request.text).tolist()
    return {"vector": vector}