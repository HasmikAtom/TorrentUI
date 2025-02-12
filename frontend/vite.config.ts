import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from "path"


// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  preview: {
    host: true,
    port: 3000,
    strictPort: true,
    proxy: {
      "/api" : {
        target: "http://backend:8080/",
        changeOrigin: true,
        rewrite: (path: any) => path.replace(/^\/api/, "") 
      } 
      
    }
  },
  server: {
    host: true,
    port: 3000,
    strictPort: true,
    proxy: {
      "/api" : {
        target: "http://backend:8080/",
        changeOrigin: true,
        rewrite: (path: any) => path.replace(/^\/api/, "") 
      } 
      
    }
  }
})
