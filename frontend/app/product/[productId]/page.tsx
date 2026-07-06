'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';

interface Product {
  id: string;
  name: string;
  price: string;
  image: string;
  category: string;
  url: string;
  description: string;
  in_stock: boolean;
  source: string;
}

export default function ProductDetail() {
  // useParams() peut renvoyer un objet où productId est parfois un tableau
  const params = useParams();
  const productId = Array.isArray(params.productId) ? params.productId[0] : params.productId;
  
  const [product, setProduct] = useState<Product | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!productId) return;

    console.log("Fetching product with ID:", productId);

    fetch(`http://localhost:8080/api/products/${productId}`)
      .then(res => {
        if (!res.ok) throw new Error(`Erreur serveur: ${res.status}`);
        return res.json();
      })
      .then(data => {
        console.log("Données reçues:", data);
        setProduct(data);
      })
      .catch(err => {
        console.error("Erreur fetch:", err);
        setError(err.message);
      });
  }, [productId]);

  if (error) return <div className="p-10 text-red-600">Erreur: {error}</div>;
  if (!product) return <div className="p-10">Chargement du produit...</div>;

  return (
    <main className="p-8 max-w-2xl mx-auto">
      <button 
        onClick={() => window.history.back()} 
        className="mb-4 text-blue-700 font-bold hover:underline"
      >
        ← Retour
      </button>
      
      <div className="bg-white p-6 rounded-xl shadow">
        <img 
          src={product.image} 
          className="w-full h-64 object-contain" 
          alt={product.name} 
        />
        <h1 className="text-2xl font-bold mt-4">{product.name}</h1>
        <p className="text-3xl text-blue-700 font-bold my-4">{product.price}</p>
        
        <div className="text-gray-600 mb-6">
          <p><strong>Catégorie:</strong> {product.category}</p>
          <p><strong>Stock:</strong> {product.in_stock ? "Oui" : "Non"}</p>
        </div>

        <a 
          href={product.url} 
          target="_blank" 
          rel="noopener noreferrer"
          className="block text-center bg-yellow-400 py-3 rounded-lg font-bold hover:bg-yellow-500 transition-colors"
        >
          🛒 Voir sur {product.source?.toUpperCase()}
        </a>
      </div>
    </main>
  );
}