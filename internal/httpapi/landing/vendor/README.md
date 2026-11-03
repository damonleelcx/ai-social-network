three.js is three@0.187.0 (MIT, see three.LICENSE), reduced to the classes scene.js imports:

    echo "export { <the names scene.js imports> } from 'three';" > entry.js
    npx esbuild entry.js --bundle --minify --format=esm --outfile=three.js
