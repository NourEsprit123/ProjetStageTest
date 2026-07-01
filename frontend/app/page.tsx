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

// REMIS À 24 POUR UNE GRILLE STANDARD PARFAITE
const PRODUCTS_PER_PAGE = 24;

export default function Home() {
  const [search, setSearch] = useState('');
  const [category, setCategory] = useState('');
  
  // Contient TOUS les produits retournés par le scraper
  const [allProducts, setAllProducts] = useState<Product[]>([]); 
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
    setCurrentPage(1); 
    
    try {
      const params = new URLSearchParams();
      if (search) params.append('search', search);
      if (category) params.append('category', category);

      const res = await fetch(`http://localhost:8080/api/products?${params}`);
      const data = await res.json();
      
      // Sécurité pour forcer la capture de TOUS les produits envoyés par Go
      if (data && data.products && Array.isArray(data.products)) {
        setAllProducts(data.products);
      } else if (Array.isArray(data)) {
        setAllProducts(data);
      } else {
        setAllProducts([]);
      }
    } catch (err) {
      console.error("Erreur de récupération :", err);
    } finally {
      setLoading(false);
    }
  };

  // --- LOGIQUE DE PAGINATION CÔTÉ CLIENT ---
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
        <h1 className="text-3xl font-bold mb-2">🛍️ Multi-Source Scraper</h1>
        <p className="text-blue-200">Tunisianet • Mytek • Wiki.tn</p>
      </div>

      {/* Search Section */}
      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="bg-white rounded-xl shadow p-6 flex flex-col gap-4">
          <input
            type="text"
            placeholder="Rechercher un produit..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            className="border border-gray-300 rounded-lg px-4 py-3 text-lg focus:outline-none focus:border-blue-500 text-gray-800"
          />

          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="border border-gray-300 rounded-lg px-4 py-3 text-lg focus:outline-none focus:border-blue-500 text-gray-800"
          >
            <option value="">Toutes les catégories</option>
            {categories.map((cat) => (
              <option key={cat.value} value={cat.value}>
                {cat.label}
              </option>
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
            <div className="text-center py-12 text-blue-700 text-xl font-medium">
              ⏳ Extraction en direct depuis Tunisianet, Mytek et Wiki...
            </div>
          )}

          {!loading && searched && allProducts.length === 0 && (
            <div className="text-center py-12 text-gray-500 text-xl">
              ❌ Aucun produit trouvé
            </div>
          )}

          {!loading && allProducts.length > 0 && (
            <>
              {/* Vrai compteur global lié à allProducts.length */}
              <p className="text-gray-600 mb-4 font-medium">
                {allProducts.length} produits trouvés — Page {currentPage} sur {totalPages}
              </p>

              {/* Grid des produits de la page courante */}
              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6">
                {displayedProducts.map((product) => (
                  <div 
                    key={product.id} 
                    className="bg-white rounded-xl shadow hover:shadow-lg transition p-4 flex flex-col justify-between cursor-pointer border border-gray-100"
                    onClick={() => router.push(`/product/${product.id}?search=${encodeURIComponent(search)}&category=${encodeURIComponent(category)}`)}
                  >
                    <div>
                      <div className="w-full h-48 relative bg-gray-50 rounded-lg overflow-hidden mb-4">
                        <img 
                          src={product.image || 'https://via.placeholder.com/150'} 
                          alt={product.name} 
                          className="w-full h-full object-contain mix-blend-multiply"
                        />
                      </div>
                      <h2 className="font-bold text-gray-800 text-sm line-clamp-2 mb-2 h-10">{product.name}</h2>
                    </div>
                    
                    <div className="mt-4">
                      <div className="flex justify-between items-center mb-3">
                        <span className="text-lg font-extrabold text-blue-700">{product.price}</span>
                        <span className={`text-xs px-2 py-1 rounded-full font-semibold ${
                          product.id.startsWith('wiki') ? 'bg-purple-100 text-purple-800' :
                          product.id.startsWith('mytek') ? 'bg-orange-100 text-orange-800' : 'bg-blue-100 text-blue-800'
                        }`}>
                          {product.id.startsWith('wiki') ? 'Wiki.tn' : product.id.startsWith('mytek') ? 'Mytek' : 'Tunisianet'}
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
                ))}
              </div>

              {/* Pagination UI */}
              {totalPages > 1 && (
                <div className="flex items-center justify-center gap-2 mt-12 pb-12">
                  <button
                    onClick={() => goToPage(currentPage - 1)}
                    disabled={currentPage === 1}
                    className="px-3 py-2 rounded-lg border border-gray-300 text-gray-600 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed font-medium text-sm transition"
                  >
                    « Précédent
                  </button>

                  {getPageNumbers().map((page, idx) => (
                    <button
                      key={idx}
                      onClick={() => typeof page === 'number' && goToPage(page)}
                      disabled={page === '...'}
                      className={`px-4 py-2 rounded-lg border font-semibold text-sm transition ${
                        page === currentPage
                          ? 'bg-blue-700 border-blue-700 text-white'
                          : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50 disabled:border-none disabled:bg-transparent'
                      }`}
                    >
                      {page}
                    </button>
                  ))}

                  <button
                    onClick={() => goToPage(currentPage + 1)}
                    disabled={currentPage === totalPages}
                    className="px-3 py-2 rounded-lg border border-gray-300 text-gray-600 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed font-medium text-sm transition"
                  >
                    Suivant »
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