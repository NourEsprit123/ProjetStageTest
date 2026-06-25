'use client';

import { useEffect, useState, Suspense } from 'react';
import { useParams, useSearchParams, useRouter } from 'next/navigation';

interface Product {
  id: string;
  name: string;
  price: string;
  image: string;
  category: string;
  url: string;
  description: string;
  in_stock: boolean;
}

function ProductDetailContent() {
  const params = useParams();
  const searchParams = useSearchParams();
  const router = useRouter();
  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const search = searchParams.get('search') || '';
  const category = searchParams.get('category') || '';

  useEffect(() => {
    const fetchProduct = async () => {
      try {
        const query = new URLSearchParams();
        if (search) query.append('search', search);
        if (category) query.append('category', category);
        const res = await fetch(`http://localhost:8080/api/products/${params.productId}?${query}`);
        const data = await res.json();
        setProduct(data);
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    };
    fetchProduct();
  }, [params.productId, search, category]);

  if (loading) {
    return <div className="min-h-screen flex items-center justify-center text-blue-700 text-xl">⏳ Chargement...</div>;
  }

  if (!product || (product as any).error) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center gap-4">
        <p className="text-gray-500 text-xl">❌ Produit non trouvé</p>
        <button onClick={() => router.back()} className="bg-blue-700 text-white px-6 py-2 rounded-lg">
          ← Retour
        </button>
      </div>
    );
  }

  return (
    <main className="min-h-screen bg-gray-50">
      <div className="bg-blue-700 text-white py-6 px-4">
        <div className="max-w-4xl mx-auto">
          <button onClick={() => router.back()} className="text-blue-200 hover:text-white mb-2">
            ← Retour aux résultats
          </button>
          <h1 className="text-2xl font-bold">Détail du produit</h1>
        </div>
      </div>
      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="bg-white rounded-xl shadow p-6">
          <div className="flex flex-col md:flex-row gap-8">
            {product.image && (
              <div className="md:w-1/2">
                <img src={product.image} alt={product.name} className="w-full object-contain rounded-lg max-h-80" />
              </div>
            )}
            <div className="md:w-1/2 flex flex-col gap-4">
              <h2 className="text-xl font-bold text-gray-800">{product.name}</h2>
              <p className="text-3xl font-bold text-blue-700">{product.price}</p>
              {product.category && (
                <span className="bg-blue-100 text-blue-700 px-3 py-1 rounded-full text-sm w-fit">
                  {product.category}
                </span>
              )}
              <div className="flex items-center gap-2">
                <span className={`w-3 h-3 rounded-full ${product.in_stock ? 'bg-green-500' : 'bg-red-500'}`}></span>
                <span className={product.in_stock ? 'text-green-600' : 'text-red-600'}>
                  {product.in_stock ? 'En stock' : 'Hors stock'}
                </span>
              </div>
              {product.url && (
                <a href={product.url} target="_blank" rel="noopener noreferrer" className="bg-yellow-400 text-gray-900 font-bold py-3 px-6 rounded-lg text-center hover:bg-yellow-500 transition">
                  🛒 Voir sur Tunisianet
                </a>
              )}
            </div>
          </div>
          {product.description && (
            <div className="mt-6 border-t pt-6">
              <h3 className="font-bold text-gray-700 mb-2">Description</h3>
              <p className="text-gray-600">{product.description}</p>
            </div>
          )}
        </div>
      </div>
    </main>
  );
}

export default function ProductDetail() {
  return (
    <Suspense fallback={<div className="min-h-screen flex items-center justify-center">⏳ Chargement...</div>}>
      <ProductDetailContent />
    </Suspense>
  );
}