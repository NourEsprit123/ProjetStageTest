'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';

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

export default function Home() {
  const [search, setSearch] = useState('');
  const [category, setCategory] = useState('');
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const router = useRouter();

  const categories = [
  { value: 'informatique', label: 'Informatique' },
  { value: 'composants', label: 'Composants' },
  { value: 'ordinateurs', label: 'Ordinateurs' },
  { value: 'reseaux', label: 'Réseaux et connectivité' },
  { value: 'peripheriques', label: 'Périphériques' },
  { value: 'stockage', label: 'Stockages' },
  { value: 'telephonie portables', label: 'Téléphones portables' },
  { value: 'smartphones', label: 'Smartphones' },
  { value: 'accessoires telephonie', label: 'Accessoires téléphonie' },
  { value: 'telephones fixes', label: 'Téléphones fixes' },
  { value: 'smartwatch', label: 'Smartwatch' },
  { value: 'sante beaute', label: 'Santé & Beauté' },
  { value: 'toiletries', label: 'Toiletries' },
  { value: 'moniteurs sante', label: 'Moniteurs de santé' },
  { value: 'bebe enfants', label: 'Bébé & enfants' },
  { value: 'pharmaceutiques', label: 'Pharmaceutiques & médicaments' },
  { value: 'soins personnels', label: 'Produits pour soins personnels' },
  { value: 'electromenager', label: 'Électroménager' },
  { value: 'aspirateurs', label: 'Aspirateurs' },
  { value: 'machine a laver', label: 'Machine à laver' },
  { value: 'seche linge', label: 'Sèche-linge' },
  { value: 'lave vaisselle', label: 'Lave-vaisselle' },
  { value: 'fours', label: 'Fours' },
];

  const handleSearch = async () => {
    if (!search && !category) return;
    setLoading(true);
    setSearched(true);
    try {
      const params = new URLSearchParams();
      if (search) params.append('search', search);
      if (category) params.append('category', category);
      const res = await fetch(`http://localhost:8080/api/products?${params}`);
      const data = await res.json();
      setProducts(data.products || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-blue-700 text-white py-8 px-4 text-center">
        <h1 className="text-3xl font-bold mb-2">🛍️ Tunisianet Scraper</h1>
        <p className="text-blue-200">Recherchez les meilleurs produits</p>
      </div>

      {/* Search Section */}
      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="bg-white rounded-xl shadow p-6 flex flex-col gap-4">
          {/* Search Input */}
          <input
            type="text"
            placeholder="Rechercher un produit..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            className="border border-gray-300 rounded-lg px-4 py-3 text-lg focus:outline-none focus:border-blue-500"
          />

          {/* Category Filter */}
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="border border-gray-300 rounded-lg px-4 py-3 text-lg focus:outline-none focus:border-blue-500"
          >
            <option value="">Toutes les catégories</option>
            {categories.map((cat) => (
              <option key={cat.value} value={cat.value}>
                {cat.label}
              </option>
            ))}
          </select>

          {/* Search Button */}
          <button
            onClick={handleSearch}
            className="bg-blue-700 text-white py-3 rounded-lg text-lg font-semibold hover:bg-blue-800 transition"
          >
            🔍 Rechercher
          </button>
        </div>

        {/* Results */}
        <div className="mt-8">
          {loading && (
            <div className="text-center py-12 text-blue-700 text-xl">
              ⏳ Chargement des produits...
            </div>
          )}

          {!loading && searched && products.length === 0 && (
            <div className="text-center py-12 text-gray-500 text-xl">
              ❌ Aucun produit trouvé
            </div>
          )}

          {!loading && products.length > 0 && (
            <>
              <p className="text-gray-600 mb-4">{products.length} produits trouvés</p>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
                {products.map((product) => (
                  <div
                    key={product.id}
                    onClick={() => router.push(`/product/${product.id}?search=${search}&category=${category}`)}
                    className="bg-white rounded-xl shadow hover:shadow-lg transition cursor-pointer overflow-hidden"
                  >
                    {product.image && (
                      <img
                        src={product.image}
                        alt={product.name}
                        className="w-full h-48 object-contain p-4"
                      />
                    )}
                    <div className="p-4">
                      <h2 className="font-semibold text-gray-800 text-sm line-clamp-2 mb-2">
                        {product.name}
                      </h2>
                      <p className="text-blue-700 font-bold text-lg">{product.price}</p>
                    </div>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>
      </div>
    </main>
  );
}