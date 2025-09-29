import { Routes, Route } from "react-router-dom";
import WhatsAppConversations from "./whatsapp/Conversations";
import WhatsAppConnection from "./whatsapp/WhatsAppConnection";

export default function WhatsApp() {
  return (
    <Routes>
      <Route path="/" element={<WhatsAppConversations />} />
      <Route path="/conversations" element={<WhatsAppConversations />} />
      <Route path="/connection" element={<WhatsAppConnection />} />
    </Routes>
  );
}