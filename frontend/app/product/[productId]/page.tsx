'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';

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
  const params = useParams();
  const router = useRouter();
  const [product, setProduct] = useState<Product | null>(null);

  useEffect(() => {
    fetch(`http://localhost:8080/api/products/${params.productId}`)
      .then(res => res.json())
      .then(data => setProduct(data))
      .catch(console.error);
  }, [params.productId]);

  if (!product) return <div>Chargement...</div>;

  return (
    <main className="p-8 max-w-2xl mx-auto">
      <button onClick={() => router.back()} className="mb-4 text-blue-700 font-bold">← Retour</button>
      <div className="bg-white p-6 rounded-xl shadow">
        <img src={product.image} className="w-full h-64 object-contain" />
        <h1 className="text-2xl font-bold mt-4">{product.name}</h1>
        <p className="text-3xl text-blue-700 font-bold my-4">{product.price}</p>
        
        {/* Lien vers la source réelle */}
        <a 
          href={product.url} 
          target="_blank" 
          className="block text-center bg-yellow-400 py-3 rounded-lg font-bold"
        >
          🛒 Voir sur {product.source.toUpperCase()}
        </a>
      </div>
    </main>
  );
}