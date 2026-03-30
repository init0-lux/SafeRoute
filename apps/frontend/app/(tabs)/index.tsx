import React from 'react';
import { ScrollView, TouchableOpacity } from 'react-native';
import { Shield, AlertTriangle, Map, PhoneCall, ChevronRight } from 'lucide-react-native';
import { Text, View } from '@/components/Themed';

export default function TabOneScreen() {
  return (
    <ScrollView className="flex-1 bg-gray-50 dark:bg-zinc-950">
      <View className="p-6 pt-12">
        {/* Header */}
        <View className="flex-row justify-between items-center mb-8 bg-transparent">
          <View className="bg-transparent">
            <Text className="text-3xl font-bold text-gray-900 dark:text-white">SafeRoute</Text>
            <Text className="text-gray-500 dark:text-zinc-400">Your urban safety companion</Text>
          </View>
          <TouchableOpacity className="bg-red-500 p-3 rounded-full shadow-lg">
            <PhoneCall size={24} color="white" />
          </TouchableOpacity>
        </View>

        {/* Safety Score Card */}
        <View className="bg-white dark:bg-zinc-900 p-6 rounded-3xl shadow-sm mb-6 border border-gray-100 dark:border-zinc-800">
          <View className="flex-row justify-between items-start mb-4 bg-transparent">
            <View className="bg-transparent">
              <Text className="text-lg font-semibold text-gray-700 dark:text-zinc-300">Current Safety Score</Text>
              <Text className="text-gray-400 text-sm">Based on real-time data</Text>
            </View>
            <Shield size={24} color="#10b981" />
          </View>
          <View className="flex-row items-center bg-transparent">
            <Text className="text-5xl font-black text-emerald-500">88</Text>
            <View className="ml-4 bg-transparent">
              <Text className="text-emerald-500 font-bold text-lg">Safe Area</Text>
              <Text className="text-gray-400 text-xs">Low risk detected nearby</Text>
            </View>
          </View>
        </View>

        {/* Quick Actions */}
        <Text className="text-xl font-bold mb-4 ml-1 text-gray-900 dark:text-white">Quick Actions</Text>
        <View className="flex-row flex-wrap justify-between bg-transparent">
          <QuickAction 
            icon={<AlertTriangle size={24} color="#f59e0b" />} 
            label="Report Incident" 
            sublabel="Anonymous & Verifiable"
          />
          <QuickAction 
            icon={<Map size={24} color="#3b82f6" />} 
            label="Safe Navigation" 
            sublabel="Risk-aware routing"
          />
        </View>

        {/* Recent Alerts */}
        <View className="mt-8 bg-transparent">
          <View className="flex-row justify-between items-center mb-4 bg-transparent">
            <Text className="text-xl font-bold text-gray-900 dark:text-white">Recent Alerts</Text>
            <TouchableOpacity className="bg-transparent">
              <Text className="text-blue-500 font-medium">View all</Text>
            </TouchableOpacity>
          </View>
          <AlertItem 
            title="Crowded Path" 
            location="MG Road • 200m away"
            time="5 mins ago"
            type="info"
          />
          <AlertItem 
            title="Poor Lighting" 
            location="8th Cross St • 1.2km away"
            time="15 mins ago"
            type="warning"
          />
        </View>
      </View>
    </ScrollView>
  );
}

function QuickAction({ icon, label, sublabel }: { icon: React.ReactNode, label: string, sublabel: string }) {
  return (
    <TouchableOpacity className="bg-white dark:bg-zinc-900 w-[48%] p-4 rounded-2xl mb-4 shadow-sm border border-gray-100 dark:border-zinc-800">
      <View className="mb-3 bg-transparent">{icon}</View>
      <Text className="font-bold text-gray-900 dark:text-white mb-1">{label}</Text>
      <Text className="text-gray-400 text-[10px] leading-tight">{sublabel}</Text>
    </TouchableOpacity>
  );
}

function AlertItem({ title, location, time, type }: { title: string, location: string, time: string, type: 'info' | 'warning' }) {
  return (
    <View className="flex-row items-center bg-white dark:bg-zinc-900 p-4 rounded-2xl mb-3 shadow-sm border border-gray-100 dark:border-zinc-800">
      <View className={`w-2 h-10 rounded-full mr-4 ${type === 'warning' ? 'bg-amber-400' : 'bg-blue-400'}`} />
      <View className="flex-1 bg-transparent">
        <Text className="font-bold text-gray-900 dark:text-white">{title}</Text>
        <Text className="text-gray-400 text-xs">{location}</Text>
      </View>
      <Text className="text-gray-400 text-[10px]">{time}</Text>
      <ChevronRight size={16} color="#9ca3af" style={{ marginLeft: 8 }} />
    </View>
  );
}
