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
  source: string;
}

const PRODUCTS_PER_PAGE = 24;

export default function Home() {
  const [search, setSearch] = useState('');
  const [category, setCategory] = useState('');
  const [allProducts, setAllProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const router = useRouter();

  const categories = [
    { value: 'informatique', label: 'Informatique' },
    { value: 'ordinateurs', label: 'Ordinateurs' },
    { value: 'smartphones', label: 'Smartphones' },
    { value: 'composants', label: 'Composants' },
    { value: 'reseaux', label: 'Réseaux et connectivité' },
    { value: 'peripheriques', label: 'Périphériques' },
    { value: 'stockage', label: 'Stockages' },
    { value: 'electromenager', label: 'Électroménager' },
    { value: 'telephonie portables', label: 'Téléphones portables' },
    { value: 'accessoires telephonie', label: 'Accessoires téléphonie' },
    { value: 'smartwatch', label: 'Smartwatch' },
    { value: 'machine a laver', label: 'Machine à laver' },
    { value: 'lave vaisselle', label: 'Lave-vaisselle' },
    { value: 'aspirateurs', label: 'Aspirateurs' },
    { value: 'fours', label: 'Fours' },
  ];

  const handleSearch = async () => {
    if (!search && !category) return;
    setLoading(true);
    setSearched(true);
    setCurrentPage(1);
    try {
      const params = new URLSearchParams();
      if (search) params.append('search', search);
      if (category) params.append('category', category);
      const res = await fetch(`http://localhost:8080/api/products?${params}`);
      const data = await res.json();
      setAllProducts(data.products || []);
    } catch (err) {
      console.error('Erreur:', err);
      setAllProducts([]);
    } finally {
      setLoading(false);
    }
  };

  const totalPages = Math.ceil(allProducts.length / PRODUCTS_PER_PAGE);
  const displayedProducts = allProducts.slice(
    (currentPage - 1) * PRODUCTS_PER_PAGE,
    currentPage * PRODUCTS_PER_PAGE
  );

  const goToPage = (page: number) => {
    setCurrentPage(page);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const getPageNumbers = () => {
    const pages: (number | string)[] = [];
    if (totalPages <= 7) {
      for (let i = 1; i <= totalPages; i++) pages.push(i);
      return pages;
    }
    pages.push(1);
    const start = Math.max(2, currentPage - 1);
    const end = Math.min(totalPages - 1, currentPage + 1);
    if (start > 2) pages.push('...');
    for (let i = start; i <= end; i++) pages.push(i);
    if (end < totalPages - 1) pages.push('...');
    pages.push(totalPages);
    return pages;
  };

  const getBadge = (source: string) => {
    if (source === 'wiki') return { label: 'Wiki.tn', cls: 'bg-purple-100 text-purple-800' };
    if (source === 'mytek') return { label: 'Mytek', cls: 'bg-orange-100 text-orange-800' };
    return { label: 'Tunisianet', cls: 'bg-blue-100 text-blue-800' };
  };

  return (
    <main className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-blue-700 text-white py-8 px-4 text-center">
        <h1 className="text-3xl font-bold mb-1">🛍️ Multi-Source Scraper</h1>
        <p className="text-blue-200 text-sm">Tunisianet • Mytek • Wiki.tn</p>
      </div>

      {/* Search */}
      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="bg-white rounded-xl shadow p-6 flex flex-col gap-4">
          <input
            type="text"
            placeholder="Rechercher un produit..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            className="border border-gray-300 rounded-lg px-4 py-3 text-lg focus:outline-none focus:border-blue-500"
          />
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="border border-gray-300 rounded-lg px-4 py-3 text-lg focus:outline-none focus:border-blue-500"
          >
            <option value="">Toutes les catégories</option>
            {categories.map((cat) => (
              <option key={cat.value} value={cat.value}>{cat.label}</option>
            ))}
          </select>
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
              ⏳ Chargement...
            </div>
          )}

          {!loading && searched && allProducts.length === 0 && (
            <div className="text-center py-12 text-gray-500 text-xl">
              ❌ Aucun produit trouvé
            </div>
          )}

          {!loading && allProducts.length > 0 && (
            <>
              <p className="text-gray-600 mb-4 font-medium">
                {allProducts.length} produits trouvés — Page {currentPage} sur {totalPages}
              </p>

              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6">
                {displayedProducts.map((product, idx) => {
                  const badge = getBadge(product.source);
                  return (
                    <div
                      key={product.id || `product-${idx}`}
                      className="bg-white rounded-xl shadow hover:shadow-lg transition cursor-pointer overflow-hidden border border-gray-100"
                      onClick={() => router.push(`/product/${encodeURIComponent(product.id)}?search=${encodeURIComponent(search)}&category=${encodeURIComponent(category)}`)}
                    >
                      {product.image && (
                        <img
                          src={product.image}
                          alt={product.name}
                          className="w-full h-48 object-contain p-4"
                        />
                      )}
                      <div className="p-4">
                        <h2 className="font-semibold text-gray-800 text-sm line-clamp-2 mb-3">
                          {product.name}
                        </h2>
                        <div className="flex justify-between items-center mb-2">
                          <span className="text-blue-700 font-bold text-lg">{product.price}</span>
                          <span className={`text-xs px-2 py-1 rounded-full font-semibold ${badge.cls}`}>
                            {badge.label}
                          </span>
                        </div>
                        <div className="flex items-center gap-1.5 text-xs">
                          <span className={`w-2.5 h-2.5 rounded-full ${product.in_stock ? 'bg-green-500' : 'bg-red-500'}`}></span>
                          <span className={product.in_stock ? 'text-green-600 font-medium' : 'text-red-500 font-medium'}>
                            {product.in_stock ? 'En Stock' : 'Rupture'}
                          </span>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex justify-center items-center gap-2 mt-8 flex-wrap">
                  <button
                    onClick={() => goToPage(currentPage - 1)}
                    disabled={currentPage === 1}
                    className="px-4 py-2 rounded-lg border border-gray-300 disabled:opacity-40 disabled:cursor-not-allowed hover:bg-gray-100"
                  >
                    ← Précédent
                  </button>
                  {getPageNumbers().map((p, idx) =>
                    p === '...' ? (
                      <span key={`dots-${idx}`} className="px-2 text-gray-400">…</span>
                    ) : (
                      <button
                        key={p}
                        onClick={() => goToPage(p as number)}
                        className={`px-4 py-2 rounded-lg border ${
                          currentPage === p
                            ? 'bg-blue-700 text-white border-blue-700'
                            : 'border-gray-300 hover:bg-gray-100'
                        }`}
                      >
                        {p}
                      </button>
                    )
                  )}
                  <button
                    onClick={() => goToPage(currentPage + 1)}
                    disabled={currentPage === totalPages}
                    className="px-4 py-2 rounded-lg border border-gray-300 disabled:opacity-40 disabled:cursor-not-allowed hover:bg-gray-100"
                  >
                    Suivant →
                  </button>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </main>
  );
}
