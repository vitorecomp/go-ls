import './globals.css';

export const metadata = {
  title: 'go-ls — Filesystem Snapshot, Merkle Hash & Diff Console',
  description:
    'Inspect directory scans, verify folder Merkle hashes, compare snapshots across hosts, and detect duplicate files.',
};

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
