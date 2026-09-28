import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: 50,          // 50 одночасних віртуальних користувачів
  duration: '10s',  // тривалість тесту 10 секунд
};

export default function () {
  // Змінюй порт (3000 або 8080) та ендпойнт (/io або /cpu) залежно від тесту
  const res = http.get('http://localhost:3000/io');
  check(res, { 'status is 200': (r) => r.status === 200 });
}