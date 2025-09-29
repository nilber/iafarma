import { Routes, Route } from "react-router-dom";
import SalesOrders from "./sales/Orders";
import SalesDashboard from "./sales/Dashboard";
import OrderDetails from "./sales/OrderDetails";
import PaymentMethods from "./sales/PaymentMethods";

export default function Sales() {
  return (
    <Routes>
      <Route path="/" element={<SalesOrders />} />
      <Route path="/orders" element={<SalesOrders />} />
      <Route path="/orders/:id" element={<OrderDetails />} />
      <Route path="/dashboard" element={<SalesDashboard />} />
      <Route path="/payment-methods" element={<PaymentMethods />} />
    </Routes>
  );
}