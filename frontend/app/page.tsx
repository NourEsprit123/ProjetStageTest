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

const PRODUCTS_PER_PAGE = 24;

export default function Home() {
  const [search, setSearch] = useState('');
  const [category, setCategory] = useState('');
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
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
    setCurrentPage(1); // reset à la page 1 à chaque nouvelle recherche
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

  const totalPages = Math.ceil(products.length / PRODUCTS_PER_PAGE);
  const paginatedProducts = products.slice(
    (currentPage - 1) * PRODUCTS_PER_PAGE,
    currentPage * PRODUCTS_PER_PAGE
  );

  const goToPage = (page: number) => {
    setCurrentPage(page);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  // Génère une liste de numéros de page avec "..." si trop de pages
  const getPageNumbers = () => {
    const pages: (number | string)[] = [];
    const maxVisible = 5;

    if (totalPages <= maxVisible + 2) {
      for (let i = 1; i <= totalPages; i++) pages.push(i);
      return pages;
    }

    pages.push(1);
    let start = Math.max(2, currentPage - 1);
    let end = Math.min(totalPages - 1, currentPage + 1);

    if (start > 2) pages.push('...');
    for (let i = start; i <= end; i++) pages.push(i);
    if (end < totalPages - 1) pages.push('...');

    pages.push(totalPages);
    return pages;
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
              <p className="text-gray-600 mb-4">
                {products.length} produits trouvés — page {currentPage} sur {totalPages}
              </p>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
                {paginatedProducts.map((product) => (
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

              {/* Pagination Controls */}
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
                      <span key={`dots-${idx}`} className="px-2 text-gray-400">
                        …
                      </span>
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
