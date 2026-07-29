declare global {
  namespace App {
    interface Locals {
      user: { username: string } | null;
    }
    interface LayoutData {
      user: { username: string } | null;
      brand: BrandSettings;
    }
  }
}

interface BrandSettings {
  logo_url: string;
  brand_name: string;
}

export {};
