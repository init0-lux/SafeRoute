import React from 'react';
import { TouchableOpacity, Text, StyleSheet, View } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';

interface QuickReportButtonProps {
    onPress: () => void;
}

export default function QuickReportButton({ onPress }: QuickReportButtonProps) {
    return (
        <TouchableOpacity style={styles.container} onPress={onPress} activeOpacity={0.85}>
            <View style={styles.inner}>
                <Text style={styles.text}>quick report</Text>
                <View style={styles.iconCircle}>
                    <Ionicons name="arrow-forward" size={18} color={Colors.white} />
                </View>
            </View>
        </TouchableOpacity>
    );
}

const styles = StyleSheet.create({
    container: {
        backgroundColor: Colors.cardBg,
        borderRadius: BorderRadius.full,
        paddingVertical: Spacing.sm,
        paddingLeft: Spacing.md,
        paddingRight: Spacing.xs,
        borderWidth: 1,
        borderColor: Colors.border,
    },
    inner: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: Spacing.sm,
    },
    text: {
        fontSize: FontSizes.sm,
        color: Colors.white,
        fontWeight: '500',
    },
    iconCircle: {
        width: 32,
        height: 32,
        borderRadius: BorderRadius.full,
        backgroundColor: Colors.grayDark,
        justifyContent: 'center',
        alignItems: 'center',
    },
});
