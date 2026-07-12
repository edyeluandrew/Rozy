/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        cream: '#FDF9ED',
        beige: '#F0E0B1',
        sand: '#EBE5D7',
        gold: '#EAC333',
        'dark-gold': '#6E5505',
        kdblack: '#020201',
        charcoal: '#2A2925',
        kdgrey: '#85867E',
        kdborder: '#C2C1BF',
        kdwhite: '#FCFCFB',
      },
    },
  },
  plugins: [],
}
