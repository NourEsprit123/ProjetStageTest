'use client';

import { useState, useRef,useEffect } from 'react';
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
  const eventSourceRef = useRef<EventSource | null>(null);
  const router = useRouter();


const [scraping, setScraping] = useState(false);
const [scrapeMessage, setScrapeMessage] = useState('');



const handleTriggerScraping = async () => {
  setScraping(true);
  setScrapeMessage('');
  try {
    const res = await fetch('http://localhost:8080/api/scrape', { method: 'POST' });
    const data = await res.json();
    setScrapeMessage(data.message || 'Scraping lancé.');
  } catch {
    setScrapeMessage('Erreur lors du déclenchement.');
  } finally {
    setScraping(false);
  }
};

  // 👈 AJOUTEZ LE useEffect ICI
  useEffect(() => {
  // Suppression du setTimeout pour une exécution instantanée
  if (search.length >= 3) {
    handleSearch();
  } else if (search === "") {
    setAllProducts([]); 
  }
}, [search, category]);

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
    setLoading(true);
    setSearched(true);
    setAllProducts([]);
    setCurrentPage(1); // reset pagination à chaque nouvelle recherche

    const params = new URLSearchParams();
    if (search) params.append('search', search);
    if (category) params.append('category', category);

    try {
        const res = await fetch(`http://localhost:8080/api/products?${params}`);
        const data = await res.json();
        setAllProducts(data.products || []);
    } catch (err) {
        console.error("Erreur lors de la récupération :", err);
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

  return (
    <main className="min-h-screen bg-gray-50">
    <div className="bg-blue-700 text-white py-8 px-4 text-center">
  <h1 className="text-3xl font-bold mb-2">🛍️ Multi-Source Scraper</h1>
  <button
    onClick={handleTriggerScraping}
    disabled={scraping}
    className="mt-2 bg-white text-blue-700 px-4 py-2 rounded-lg font-semibold disabled:opacity-50"
  >
    {scraping ? '⏳ Scraping en cours...' : '🔄 Lancer le scraping'}
  </button>
  {scrapeMessage && <p className="text-sm mt-2 text-blue-100">{scrapeMessage}</p>}
</div>

      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="bg-white rounded-xl shadow p-6 flex flex-col gap-4">
          <input
            type="text"
            placeholder="Rechercher un produit..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="border border-gray-300 rounded-lg px-4 py-3"
          />

          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="border border-gray-300 rounded-lg px-4 py-3 bg-white"
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
            className="bg-blue-700 text-white py-3 rounded-lg font-semibold"
          >
            {loading ? 'Recherche en cours...' : '🔍 Rechercher'}
          </button>
        </div>

        <div className="mt-8">
          {loading && (
             <div className="text-center py-4 text-blue-600 animate-pulse">
               Chargement en temps réel... {allProducts.length} produits trouvés pour l'instant.
             </div>
          )}

          {!loading && searched && (
            <div className="text-sm text-gray-500 mb-4">
              {allProducts.length} produit{allProducts.length > 1 ? 's' : ''} trouvé{allProducts.length > 1 ? 's' : ''}
            </div>
          )}

          {/* div grid restaurée (elle manquait dans le fichier collé) */}
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6">
            {displayedProducts.map((product) => (
              <div key={product.id} className="bg-white p-4 rounded-lg shadow relative">
                {product.source && (
                  <span className="absolute top-2 right-2 text-xs bg-gray-800 text-white px-2 py-1 rounded-full">
                    {product.source}
                  </span>
                )}

                <span
    className={`absolute top-2 left-2 text-xs px-2 py-1 rounded-full font-semibold ${
      product.in_stock
        ? 'bg-green-100 text-green-700'
        : 'bg-red-100 text-red-700'
    }`}
  >
    {product.in_stock ? 'En stock' : 'Rupture'}
  </span>
                <img src={product.image} className="w-full h-40 object-contain" />
                <h2 className="text-sm font-bold mt-2">{product.name}</h2>
                <p className="text-blue-700 font-bold">{product.price}</p>
              </div>
            ))}
          </div>

          {!loading && totalPages > 1 && (
            <div className="flex justify-center items-center gap-2 mt-10 flex-wrap">
              <button
                onClick={() => goToPage(Math.max(1, currentPage - 1))}
                disabled={currentPage === 1}
                className="px-4 py-2 bg-blue-700 text-white rounded-lg disabled:opacity-40 disabled:cursor-not-allowed"
              >
                ← Précédent
              </button>

              <div className="flex gap-1">
                {Array.from({ length: totalPages }, (_, i) => i + 1)
                  .filter(
                    (page) =>
                      page === 1 ||
                      page === totalPages ||
                      Math.abs(page - currentPage) <= 1
                  )
                  .map((page, idx, arr) => (
                    <span key={page} className="flex items-center">
                      {idx > 0 && arr[idx - 1] !== page - 1 && (
                        <span className="px-2 text-gray-400">…</span>
                      )}
                      <button
                        onClick={() => goToPage(page)}
                        className={`px-3 py-2 rounded-lg ${
                          page === currentPage
                            ? 'bg-blue-700 text-white font-bold'
                            : 'bg-white border border-gray-300 text-gray-700 hover:bg-gray-100'
                        }`}
                      >
                        {page}
                      </button>
                    </span>
                  ))}
              </div>

              <button
                onClick={() => goToPage(Math.min(totalPages, currentPage + 1))}
                disabled={currentPage === totalPages}
                className="px-4 py-2 bg-blue-700 text-white rounded-lg disabled:opacity-40 disabled:cursor-not-allowed"
              >
                Suivant →
              </button>
            </div>
          )}

          {!loading && searched && allProducts.length === 0 && (
            <div className="text-center py-10 text-gray-500">
              Aucun produit trouvé.
            </div>
          )}
        </div>
      </div>
    </main>
  );
}