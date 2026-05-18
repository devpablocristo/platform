/* eslint-disable react-refresh/only-export-components -- hooks acoplados al Context del mismo archivo */
import { createContext, useCallback, useContext, useEffect, useRef, useState, type PropsWithChildren } from 'react';

type PageSearchContextValue = {
  query: string;
  setQuery: (value: string) => void;
  register: () => () => void;
  visible: boolean;
  placeholder: string;
};

const PageSearchContext = createContext<PageSearchContextValue>({
  query: '',
  setQuery: () => {},
  register: () => () => {},
  visible: false,
  placeholder: 'Search...',
});

const PageSearchShellContext = createContext(false);

/**
 * Hook que registra la página como consumidora del search y devuelve el query.
 * Al desmontar, des-registra y el input desaparece.
 */
export function usePageSearch(): string {
  const { query, register } = useContext(PageSearchContext);
  useEffect(() => register(), [register]);
  return query;
}

/**
 * Hook para el Shell: controla visibilidad y query del search input.
 */
export function usePageSearchShellControl() {
  const { query, setQuery, visible, placeholder } = useContext(PageSearchContext);
  return {
    query,
    visible,
    placeholder,
    setQuery,
    clear: () => setQuery(''),
  };
}

/**
 * Provider del buscador de página. Se monta una vez en el Shell.
 * Las páginas se registran con usePageSearch() — cuando hay al menos una registrada, el input es visible.
 */
export function PageSearchProvider({
  children,
  placeholder = 'Search...',
}: PropsWithChildren<{ placeholder?: string }>) {
  const [query, setQuery] = useState('');
  const countRef = useRef(0);
  const [visible, setVisible] = useState(false);

  const register = useCallback(() => {
    countRef.current += 1;
    setVisible(true);
    return () => {
      countRef.current -= 1;
      if (countRef.current <= 0) {
        countRef.current = 0;
        setVisible(false);
        setQuery('');
      }
    };
  }, []);

  return (
    <PageSearchShellContext.Provider value>
      <PageSearchContext.Provider value={{ query, setQuery, register, visible, placeholder }}>
        {children}
      </PageSearchContext.Provider>
    </PageSearchShellContext.Provider>
  );
}
