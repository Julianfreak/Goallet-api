import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '20s', target: 10 }, // Aumento gradual a 10 usuarios concurrentes
    { duration: '40s', target: 30 }, // Carga sostenida de 30 usuarios concurrentes
    { duration: '10s', target: 0 },  // Descenso gradual (Ramp-down)
  ],
  thresholds: {
    http_req_duration: ['p(95)<400'], // El 95% de las peticiones deben responder en menos de 400ms
    http_req_failed: ['rate<0.01'],   // La tasa de errores debe ser inferior al 1%
  },
};

// Apuntamos al nombre del contenedor interno dentro de la red Docker
const BASE_URL = 'http://goallet-api:8080/api/v1';

export default function () {
  // 1. Crear una nueva cuenta con un titular aleatorio
  const randomName = `User_${Math.floor(Math.random() * 100000)}`;
  const createPayload = JSON.stringify({ titular: randomName });
  
  const createRes = http.post(`${BASE_URL}/cuentas`, createPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  const isCreated = check(createRes, {
    'cuenta creada (201 o 200)': (r) => r.status === 201 || r.status === 200,
  });

  if (isCreated && createRes.body) {
    try {
      const data = JSON.parse(createRes.body);
      const cuentaId = data.id || data.ID;

      if (cuentaId) {
        // 2. Realizar un Depósito
        const depositRes = http.post(
          `${BASE_URL}/cuentas/${cuentaId}/depositar`,
          JSON.stringify({ monto: 1000.50 }),
          { headers: { 'Content-Type': 'application/json' } }
        );
        check(depositRes, { 'depósito exitoso': (r) => r.status === 200 });

        // 3. Realizar un Retiro
        const withdrawRes = http.post(
          `${BASE_URL}/cuentas/${cuentaId}/retirar`,
          JSON.stringify({ monto: 250.00 }),
          { headers: { 'Content-Type': 'application/json' } }
        );
        check(withdrawRes, { 'retiro exitoso': (r) => r.status === 200 });
      }
    } catch (e) {
      // Ignorar errores de parseo aislados bajo estrés extremo
    }
  }

  sleep(1);
}