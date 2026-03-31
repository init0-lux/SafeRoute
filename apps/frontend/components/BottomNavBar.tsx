import React from 'react';
import { View, TouchableOpacity, StyleSheet, Text } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import { AnimatedButton } from './AnimatedButton';

interface BottomNavBarProps {
    activeTab: string;
    onTabPress: (tab: string) => void;
}

const tabs = [
    { key: 'home', icon: 'home-outline' as const, activeIcon: 'home' as const, label: 'Home' },
    { key: 'location', icon: 'location-outline' as const, activeIcon: 'location' as const, label: 'Routes' },
    { key: 'report', icon: 'folder-outline' as const, activeIcon: 'folder' as const, label: 'Report' },
];

export default function BottomNavBar({ activeTab, onTabPress }: BottomNavBarProps) {
    return (
        <View style={styles.container}>
            <View style={styles.navBar}>
                {tabs.map((tab) => {
                    const isActive = activeTab === tab.key;
                    return (
                        <TouchableOpacity
                            key={tab.key}
                            style={[styles.tab, isActive && styles.activeTab]}
                            onPress={() => onTabPress(tab.key)}
                            activeOpacity={0.7}
                        >
                            <Ionicons
                                name={isActive ? tab.activeIcon : tab.icon}
                                size={24}
                                color={isActive ? Colors.bgDark : Colors.gray}
                            />
                        </TouchableOpacity>
                    );
                })}

            </View>
            <AnimatedButton onPress={() => onTabPress('sos')}>
                <View style={styles.sosButtonContainer}>
                    <View style={styles.sosBgCircle} />
                    <View style={styles.sosBgCircle} />
                    <View style={styles.sosButton}>
                        <Text style={styles.sosText}>SOS</Text>
                    </View>
                </View>
            </AnimatedButton >
        </View >
    );
}

const styles = StyleSheet.create({
    container: {
        position: "absolute",
        bottom: Spacing.lg,
        width: "100%",
        flexDirection: "row",
        justifyContent: "center",
        alignItems: "center",
    },
    navBar: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 24,
        backgroundColor: Colors.white,
        borderRadius: BorderRadius.full,
        paddingVertical: 8,
        paddingHorizontal: 8,
        borderWidth: 1,
        elevation: 10,
        borderColor: Colors.border,
    },
    tab: {
        width: 48,
        height: 48,
        borderRadius: BorderRadius.full,
        justifyContent: 'center',
        alignItems: 'center',
    },
    activeTab: {
        backgroundColor: Colors.purple,
    },
    sosButtonContainer: {
        position: "relative",
    },
    sosBgCircle: {
        position: "absolute",
        width: 64,
        height: 64,
        borderRadius: BorderRadius.full,
        backgroundColor: "#000",
        top: 4,
        left: 12,
    },
    sosButton: {

        width: 64,
        height: 64,
        borderRadius: BorderRadius.full,
        backgroundColor: Colors.sosRed,
        justifyContent: 'center',
        alignItems: 'center',
        marginLeft: Spacing.sm,
        borderWidth: 5,
        borderColor: "#000000ff",
    },
    sosText: {
        color: Colors.white,
        fontSize: FontSizes.sm,
        fontWeight: '800',
        letterSpacing: 1,
    },
});
